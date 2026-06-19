import Foundation

// NasGateway reaches a bound NAS's server-go API. It tries the LAN address first
// (fast, private — "直连优先") and falls back to the cloud relay when the NAS is
// not reachable on the local network. Both paths authenticate with the same
// cloud-signed device access token; the relay path additionally carries the cloud
// access token so server-cloud can authorize the account before forwarding.

struct NasConnection {
    let deviceId: String
    let deviceToken: String
    /// e.g. http://192.168.1.20:8080 — discovered on the LAN, may be nil off-LAN.
    let lanBaseURL: URL?
    /// The cloud base, e.g. https://api.higoos.cloud — used for the relay path.
    let cloudBaseURL: URL
    /// The current cloud access token, for relay authorization.
    let cloudAccessToken: String
}

enum NasGatewayError: Error {
    case unreachable
    case badStatus(Int)
    case envelope(String)
}

actor NasGateway {
    private let connection: NasConnection
    private let session: URLSession

    init(connection: NasConnection, session: URLSession = .shared) {
        self.connection = connection
        self.session = session
    }

    /// GET an `/api/v1/...` path and return the unwrapped `data` payload.
    func get<T: Decodable>(_ apiPath: String, as type: T.Type) async throws -> T {
        let data = try await request("GET", apiPath, body: nil)
        return try unwrap(data)
    }

    /// POST JSON to an `/api/v1/...` path and return the unwrapped `data` payload.
    func post<T: Decodable>(_ apiPath: String, body: Data?, as type: T.Type) async throws -> T {
        let data = try await request("POST", apiPath, body: body)
        return try unwrap(data)
    }

    // MARK: - Internals

    private func request(_ method: String, _ apiPath: String, body: Data?) async throws -> Data {
        // 1) Try LAN direct.
        if let lan = connection.lanBaseURL {
            if let data = try? await perform(method, url: lan.appendingPathComponent(apiPath), body: body, relay: false) {
                return data
            }
        }
        // 2) Fall back to the cloud relay: <cloud>/d/{deviceId}/api/v1/...
        let relayURL = connection.cloudBaseURL
            .appendingPathComponent("d")
            .appendingPathComponent(connection.deviceId)
            .appendingPathComponent(apiPath)
        return try await perform(method, url: relayURL, body: body, relay: true)
    }

    private func perform(_ method: String, url: URL, body: Data?, relay: Bool) async throws -> Data {
        var req = URLRequest(url: url)
        req.httpMethod = method
        if let body {
            req.httpBody = body
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        if relay {
            // Cloud authorizes the account, then rewrites X-Device-Token into the
            // upstream Authorization header before forwarding to the NAS.
            req.setValue("Bearer \(connection.cloudAccessToken)", forHTTPHeaderField: "Authorization")
            req.setValue(connection.deviceToken, forHTTPHeaderField: "X-Device-Token")
        } else {
            req.setValue("Bearer \(connection.deviceToken)", forHTTPHeaderField: "Authorization")
        }
        let (data, response) = try await session.data(for: req)
        guard let http = response as? HTTPURLResponse else { throw NasGatewayError.unreachable }
        guard (200..<300).contains(http.statusCode) else { throw NasGatewayError.badStatus(http.statusCode) }
        return data
    }

    /// Unwrap a server-go envelope {ok,data,error}.
    private func unwrap<T: Decodable>(_ data: Data) throws -> T {
        struct Envelope: Decodable {
            let ok: Bool
            let error: Err?
            struct Err: Decodable { let code: String; let message: String }
        }
        let meta = try JSONDecoder().decode(Envelope.self, from: data)
        if let err = meta.error { throw NasGatewayError.envelope(err.message) }
        // Decode `data` generically by re-reading the JSON.
        struct Wrapper<U: Decodable>: Decodable { let data: U }
        return try JSONDecoder().decode(Wrapper<T>.self, from: data).data
    }
}
