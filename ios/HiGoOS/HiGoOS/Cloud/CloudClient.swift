import Foundation

// CloudClient talks to server-cloud (the HiGoOS cloud control plane): it logs the
// user into their cloud account (phone-SMS / email / Apple / WeChat), persists the
// access + refresh tokens, lists the devices bound to the account, and mints the
// short-lived device access tokens the App presents to a NAS.
//
// It is the App's replacement for "no backend": once the user is signed in and a
// NAS is bound, NasGateway uses these tokens to reach the device's server-go API
// directly (LAN) or via the cloud relay (off-LAN).

// MARK: - Wire shapes (mirror server-cloud's {ok,data,error,requestId} envelope)

struct CloudEnvelope<T: Decodable>: Decodable {
    let ok: Bool
    let data: T?
    let error: CloudError?
}

struct CloudError: Decodable, Error, LocalizedError {
    let code: String
    let message: String
    var errorDescription: String? { message }
}

struct CloudUser: Decodable, Identifiable, Equatable {
    let id: String
    let displayName: String
    let avatar: String?
}

struct CloudSessionDTO: Decodable {
    let user: CloudUser
    let accessToken: String
    let refreshToken: String
}

struct BoundDeviceDTO: Decodable, Identifiable, Equatable {
    struct Inner: Decodable, Equatable {
        let id: String
        let serial: String
        let model: String
        let version: String
        let online: Bool
    }
    let device: Inner
    var id: String { device.id }
}

struct DeviceTicketDTO: Decodable {
    let deviceToken: String
    let device: BoundDeviceDTO.Inner
}

// MARK: - Token storage

/// Stores cloud tokens. Production should back this with the Keychain; this
/// scaffold uses UserDefaults so the wiring is complete and testable.
final class CloudTokenStore {
    private let defaults = UserDefaults.standard
    private let accessKey = "higo.cloud.access"
    private let refreshKey = "higo.cloud.refresh"

    var accessToken: String? {
        get { defaults.string(forKey: accessKey) }
        set { defaults.set(newValue, forKey: accessKey) }
    }
    var refreshToken: String? {
        get { defaults.string(forKey: refreshKey) }
        set { defaults.set(newValue, forKey: refreshKey) }
    }
    func clear() {
        defaults.removeObject(forKey: accessKey)
        defaults.removeObject(forKey: refreshKey)
    }
}

// MARK: - Client

actor CloudClient {
    private let baseURL: URL
    private let session: URLSession
    private let tokens: CloudTokenStore

    init(baseURL: URL, tokens: CloudTokenStore = CloudTokenStore(), session: URLSession = .shared) {
        self.baseURL = baseURL
        self.tokens = tokens
        self.session = session
    }

    var isSignedIn: Bool { tokens.refreshToken != nil }

    // MARK: Authentication

    func startSMS(phone: String) async throws {
        let _: EmptyData = try await post("/v1/auth/sms/start", body: ["phone": phone], authed: false)
    }

    func verifySMS(phone: String, code: String) async throws -> CloudUser {
        try await authenticate("/v1/auth/sms/verify", body: ["phone": phone, "code": code])
    }

    func registerEmail(email: String, password: String) async throws -> CloudUser {
        try await authenticate("/v1/auth/email/register", body: ["email": email, "password": password])
    }

    func loginEmail(email: String, password: String) async throws -> CloudUser {
        try await authenticate("/v1/auth/email/login", body: ["email": email, "password": password])
    }

    func loginApple(identityToken: String) async throws -> CloudUser {
        try await authenticate("/v1/auth/apple", body: ["identityToken": identityToken])
    }

    func loginWeChat(code: String) async throws -> CloudUser {
        try await authenticate("/v1/auth/wechat", body: ["code": code])
    }

    func logout() async {
        if let refresh = tokens.refreshToken {
            let _: EmptyData? = try? await post("/v1/auth/logout", body: ["refreshToken": refresh], authed: false)
        }
        tokens.clear()
    }

    // MARK: Devices & bindings

    func bindings() async throws -> [BoundDeviceDTO] {
        struct List: Decodable { let devices: [BoundDeviceDTO] }
        let list: List = try await get("/v1/bindings")
        return list.devices
    }

    /// Claim a pairing code shown on the NAS (binding flow A).
    func claimPairing(code: String) async throws -> DeviceTicketDTO {
        try await post("/v1/bindings/pairing/claim", body: ["code": code], authed: true)
    }

    /// Confirm a LAN-discovered device (binding flow B).
    func lanConfirm(deviceId: String) async throws -> DeviceTicketDTO {
        try await post("/v1/bindings/lan-confirm", body: ["deviceId": deviceId], authed: true)
    }

    /// Bind by serial + on-screen PIN (binding flow C).
    func bindSerial(serial: String, pin: String) async throws -> DeviceTicketDTO {
        try await post("/v1/bindings/serial", body: ["serial": serial, "pin": pin], authed: true)
    }

    /// Mint a fresh device access token for a bound device.
    func accessTicket(deviceId: String) async throws -> DeviceTicketDTO {
        try await post("/v1/devices/\(deviceId)/access-ticket", body: [:], authed: true)
    }

    func registerPush(token: String) async throws {
        let _: EmptyData = try await post("/v1/push/register", body: ["token": token, "platform": "ios"], authed: true)
    }

    // MARK: - Internals

    private struct EmptyData: Decodable {}

    private func authenticate(_ path: String, body: [String: String]) async throws -> CloudUser {
        let session: CloudSessionDTO = try await post(path, body: body, authed: false)
        tokens.accessToken = session.accessToken
        tokens.refreshToken = session.refreshToken
        return session.user
    }

    /// Returns a valid access token, refreshing once if the current one is absent.
    private func validAccessToken() async throws -> String {
        if let access = tokens.accessToken { return access }
        guard let refresh = tokens.refreshToken else { throw CloudError(code: "unauthorized", message: "not signed in") }
        let session: CloudSessionDTO = try await post("/v1/auth/refresh", body: ["refreshToken": refresh], authed: false)
        tokens.accessToken = session.accessToken
        tokens.refreshToken = session.refreshToken
        return session.accessToken
    }

    private func get<T: Decodable>(_ path: String) async throws -> T {
        var req = URLRequest(url: baseURL.appendingPathComponent(path))
        req.httpMethod = "GET"
        req.setValue("Bearer \(try await validAccessToken())", forHTTPHeaderField: "Authorization")
        return try await send(req)
    }

    private func post<T: Decodable>(_ path: String, body: [String: String], authed: Bool) async throws -> T {
        var req = URLRequest(url: baseURL.appendingPathComponent(path))
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = try JSONSerialization.data(withJSONObject: body)
        if authed {
            req.setValue("Bearer \(try await validAccessToken())", forHTTPHeaderField: "Authorization")
        }
        return try await send(req)
    }

    private func send<T: Decodable>(_ req: URLRequest) async throws -> T {
        let (data, response) = try await session.data(for: req)
        let envelope = try JSONDecoder().decode(CloudEnvelope<T>.self, from: data)
        if let err = envelope.error { throw err }
        guard let value = envelope.data else {
            let status = (response as? HTTPURLResponse)?.statusCode ?? 0
            throw CloudError(code: "decode_error", message: "empty response (status \(status))")
        }
        return value
    }
}
