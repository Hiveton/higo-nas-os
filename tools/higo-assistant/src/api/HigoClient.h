#pragma once

#include <QJsonObject>
#include <QObject>
#include <QString>
#include <functional>

class QNetworkAccessManager;
class QNetworkReply;

// Thin REST client for a single HiGoOS device. Unwraps the standard response
// envelope {ok,data,error,requestId} and hands back `data` (or an error string).
//
// Endpoints mirror docs/desktop-scanner.md §5. Network-config endpoints are
// planned backend work; the client is wired for them so the UI lights up as
// soon as the backend lands.
class HigoClient : public QObject {
    Q_OBJECT
public:
    // ok=true -> `data` holds the unwrapped payload; ok=false -> `error` is set.
    using Callback = std::function<void(bool ok, const QJsonObject &data, const QString &error)>;

    explicit HigoClient(QObject *parent = nullptr);

    void setApiBase(const QString &apiBase); // e.g. "http://10.0.0.5:8080"
    void setSessionCookie(const QString &cookie) { m_sessionCookie = cookie; }
    void setCsrfToken(const QString &token) { m_csrfToken = token; }

    // Unauthenticated device fingerprint (GET /api/v1/system/identity).
    void fetchIdentity(const Callback &cb);

    // POST /api/v1/auth/login -> captures higo_session cookie + CSRF on success.
    void login(const QString &username, const QString &password, const Callback &cb);

    // GET /api/v1/network/interfaces, GET /api/v1/network/config.
    void fetchInterfaces(const Callback &cb);
    void fetchNetworkConfig(const Callback &cb);

    // Two-phase change (high risk): PUT returns confirmationId + impact summary;
    // confirm applies it. See docs/security-governance.md.
    void planNetworkConfig(const QJsonObject &config, const Callback &cb);
    void confirmNetworkConfig(const QString &confirmationId, const Callback &cb);

private:
    void get(const QString &path, const Callback &cb);
    void send(const QByteArray &verb, const QString &path,
              const QJsonObject &body, const Callback &cb);
    void finish(QNetworkReply *reply, const Callback &cb);

    QNetworkAccessManager *m_nam = nullptr;
    QString m_apiBase;
    QString m_sessionCookie;
    QString m_csrfToken;
};
