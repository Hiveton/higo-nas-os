#include "discovery/UdpDiscoverer.h"

#include <QDateTime>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QNetworkInterface>
#include <QUdpSocket>
#include <QUuid>

UdpDiscoverer::UdpDiscoverer(QObject *parent) : QObject(parent) {
    m_socket = new QUdpSocket(this);
    // Bind to an ephemeral port on all interfaces so unicast replies come back
    // to us. ShareAddress keeps things friendly if another tool also listens.
    m_socket->bind(QHostAddress::AnyIPv4, 0,
                   QUdpSocket::ShareAddress | QUdpSocket::ReuseAddressHint);
    connect(m_socket, &QUdpSocket::readyRead, this, &UdpDiscoverer::readPending);

    m_scanTimer.setInterval(kScanIntervalMs);
    connect(&m_scanTimer, &QTimer::timeout, this, &UdpDiscoverer::sendDiscover);

    m_pruneTimer.setInterval(kScanIntervalMs);
    connect(&m_pruneTimer, &QTimer::timeout, this, &UdpDiscoverer::pruneStale);
}

void UdpDiscoverer::start() {
    if (m_scanTimer.isActive()) return;
    scanOnce();
    m_scanTimer.start();
    m_pruneTimer.start();
}

void UdpDiscoverer::stop() {
    m_scanTimer.stop();
    m_pruneTimer.stop();
}

void UdpDiscoverer::scanOnce() {
    sendDiscover();
}

QByteArray UdpDiscoverer::buildDiscover(const QString &nonce) {
    QJsonObject obj;
    obj[QStringLiteral("magic")] = QStringLiteral("HIGOOS/1");
    obj[QStringLiteral("type")] = QStringLiteral("discover");
    obj[QStringLiteral("nonce")] = nonce;
    obj[QStringLiteral("replyPort")] = 0; // reply to our source port
    return QJsonDocument(obj).toJson(QJsonDocument::Compact);
}

void UdpDiscoverer::sendDiscover() {
    // New nonce per round; replies carrying a stale nonce are ignored.
    m_nonce = QUuid::createUuid().toString(QUuid::Id128);
    const QByteArray payload = buildDiscover(m_nonce);

    // Send to the global broadcast and to each interface's directed broadcast,
    // which is the address that actually traverses most LAN segments.
    m_socket->writeDatagram(payload, QHostAddress::Broadcast, kDiscoveryPort);

    const auto ifaces = QNetworkInterface::allInterfaces();
    for (const QNetworkInterface &iface : ifaces) {
        const auto flags = iface.flags();
        if (!flags.testFlag(QNetworkInterface::IsUp) ||
            !flags.testFlag(QNetworkInterface::IsRunning) ||
            !flags.testFlag(QNetworkInterface::CanBroadcast) ||
            flags.testFlag(QNetworkInterface::IsLoopBack)) {
            continue;
        }
        for (const QNetworkAddressEntry &entry : iface.addressEntries()) {
            const QHostAddress bcast = entry.broadcast();
            if (!bcast.isNull() && entry.ip().protocol() == QAbstractSocket::IPv4Protocol) {
                m_socket->writeDatagram(payload, bcast, kDiscoveryPort);
            }
        }
    }
}

void UdpDiscoverer::readPending() {
    while (m_socket->hasPendingDatagrams()) {
        QByteArray buf;
        buf.resize(int(m_socket->pendingDatagramSize()));
        QHostAddress from;
        quint16 fromPort = 0;
        m_socket->readDatagram(buf.data(), buf.size(), &from, &fromPort);

        QJsonParseError perr{};
        const QJsonDocument doc = QJsonDocument::fromJson(buf, &perr);
        if (perr.error != QJsonParseError::NoError || !doc.isObject()) continue;
        const QJsonObject o = doc.object();

        if (o.value(QStringLiteral("magic")).toString() != QStringLiteral("HIGOOS/1")) continue;
        if (o.value(QStringLiteral("type")).toString() != QStringLiteral("announce")) continue;
        // Accept replies for the current round only (tolerate empty nonce from
        // unsolicited boot announcements).
        const QString nonce = o.value(QStringLiteral("nonce")).toString();
        if (!nonce.isEmpty() && nonce != m_nonce) continue;

        Device d;
        d.deviceId = o.value(QStringLiteral("deviceId")).toString();
        if (d.deviceId.isEmpty()) continue;
        d.model = o.value(QStringLiteral("model")).toString();
        d.version = o.value(QStringLiteral("version")).toString();
        d.hostname = o.value(QStringLiteral("hostname")).toString();
        d.initialized = o.value(QStringLiteral("initialized")).toBool();
        d.httpPort = o.value(QStringLiteral("httpPort")).toInt(8080);
        d.https = o.value(QStringLiteral("https")).toBool();
        d.primaryMac = o.value(QStringLiteral("primaryMac")).toString();
        d.netMode = o.value(QStringLiteral("netMode")).toString();
        d.uptimeSec = qint64(o.value(QStringLiteral("uptimeSec")).toDouble());
        for (const QJsonValue &v : o.value(QStringLiteral("addrs")).toArray()) {
            d.addrs << v.toString();
        }
        // Strip any IPv6 scope/host noise; QHostAddress gives us the bare form.
        d.sourceAddr = QHostAddress(from.toIPv4Address()).toString();
        d.lastSeen = QDateTime::currentDateTimeUtc();

        m_devices.insert(d.deviceId, d);
        emit deviceDiscovered(d);
    }
}

void UdpDiscoverer::pruneStale() {
    const QDateTime now = QDateTime::currentDateTimeUtc();
    for (auto it = m_devices.begin(); it != m_devices.end();) {
        if (it.value().lastSeen.msecsTo(now) > kStaleAfterMs) {
            const QString id = it.key();
            it = m_devices.erase(it);
            emit deviceLost(id);
        } else {
            ++it;
        }
    }
}
