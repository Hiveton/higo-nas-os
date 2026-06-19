#include "api/HigoClient.h"

#include <QJsonDocument>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QNetworkRequest>

HigoClient::HigoClient(QObject *parent) : QObject(parent) {
    m_nam = new QNetworkAccessManager(this);
}

void HigoClient::setApiBase(const QString &apiBase) {
    m_apiBase = apiBase;
    while (m_apiBase.endsWith(QLatin1Char('/'))) m_apiBase.chop(1);
}

void HigoClient::fetchIdentity(const Callback &cb) {
    get(QStringLiteral("/api/v1/system/identity"), cb);
}

void HigoClient::fetchInterfaces(const Callback &cb) {
    get(QStringLiteral("/api/v1/network/interfaces"), cb);
}

void HigoClient::fetchNetworkConfig(const Callback &cb) {
    get(QStringLiteral("/api/v1/network/config"), cb);
}

void HigoClient::login(const QString &username, const QString &password, const Callback &cb) {
    QJsonObject body;
    body[QStringLiteral("username")] = username;
    body[QStringLiteral("password")] = password;
    body[QStringLiteral("rememberDevice")] = true;
    // Capture the CSRF token the backend returns so later writes pass the
    // double-submit check (CSRF is only disabled in dev). The session cookie is
    // captured from Set-Cookie in finish().
    send("POST", QStringLiteral("/api/v1/auth/login"), body,
         [this, cb](bool ok, const QJsonObject &data, const QString &err) {
             if (ok) {
                 const QString csrf = data.value(QStringLiteral("csrfToken")).toString();
                 if (!csrf.isEmpty()) m_csrfToken = csrf;
             }
             if (cb) cb(ok, data, err);
         });
}

void HigoClient::planNetworkConfig(const QJsonObject &config, const Callback &cb) {
    send("PUT", QStringLiteral("/api/v1/network/config"), config, cb);
}

void HigoClient::confirmNetworkConfig(const QString &confirmationId, const Callback &cb) {
    QJsonObject body;
    body[QStringLiteral("confirmationId")] = confirmationId;
    send("POST", QStringLiteral("/api/v1/network/config/confirm"), body, cb);
}

void HigoClient::get(const QString &path, const Callback &cb) {
    send("GET", path, QJsonObject(), cb);
}

void HigoClient::send(const QByteArray &verb, const QString &path,
                      const QJsonObject &body, const Callback &cb) {
    if (m_apiBase.isEmpty()) {
        if (cb) cb(false, {}, QStringLiteral("no device selected"));
        return;
    }
    QNetworkRequest req{QUrl(m_apiBase + path)};
    req.setHeader(QNetworkRequest::ContentTypeHeader, QStringLiteral("application/json"));
    if (!m_sessionCookie.isEmpty()) {
        req.setRawHeader("Cookie", QStringLiteral("higo_session=%1").arg(m_sessionCookie).toUtf8());
    }
    if (!m_csrfToken.isEmpty()) {
        req.setRawHeader("X-CSRF-Token", m_csrfToken.toUtf8());
    }

    const QByteArray payload =
        body.isEmpty() ? QByteArray() : QJsonDocument(body).toJson(QJsonDocument::Compact);

    QNetworkReply *reply = m_nam->sendCustomRequest(req, verb, payload);
    connect(reply, &QNetworkReply::finished, this, [this, reply, cb]() { finish(reply, cb); });
}

void HigoClient::finish(QNetworkReply *reply, const Callback &cb) {
    reply->deleteLater();

    const QByteArray raw = reply->readAll();
    const QJsonDocument doc = QJsonDocument::fromJson(raw);
    const QJsonObject env = doc.object();

    // Transport error with no parseable envelope -> surface the Qt error.
    if (reply->error() != QNetworkReply::NoError && !env.contains(QStringLiteral("ok"))) {
        if (cb) cb(false, {}, reply->errorString());
        return;
    }

    const bool ok = env.value(QStringLiteral("ok")).toBool();
    if (!ok) {
        QString msg = QStringLiteral("request failed");
        const QJsonObject err = env.value(QStringLiteral("error")).toObject();
        if (!err.isEmpty()) {
            msg = err.value(QStringLiteral("message")).toString(msg);
            const QString code = err.value(QStringLiteral("code")).toString();
            if (!code.isEmpty()) msg = QStringLiteral("%1 (%2)").arg(msg, code);
        }
        if (cb) cb(false, {}, msg);
        return;
    }

    // On login, capture the session cookie + CSRF for subsequent calls.
    const QByteArray setCookie = reply->rawHeader("Set-Cookie");
    if (!setCookie.isEmpty()) {
        const QString s = QString::fromUtf8(setCookie);
        const int idx = s.indexOf(QStringLiteral("higo_session="));
        if (idx >= 0) {
            int start = idx + int(qstrlen("higo_session="));
            int end = s.indexOf(QLatin1Char(';'), start);
            m_sessionCookie = s.mid(start, end < 0 ? -1 : end - start);
        }
    }

    const QJsonValue data = env.value(QStringLiteral("data"));
    if (cb) {
        // Most payloads are objects; wrap scalars/arrays under "data" for the caller.
        if (data.isObject()) cb(true, data.toObject(), {});
        else cb(true, QJsonObject{{QStringLiteral("data"), data}}, {});
    }
}
