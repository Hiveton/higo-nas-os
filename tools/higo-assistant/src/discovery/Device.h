#pragma once

#include <QDateTime>
#include <QString>
#include <QStringList>

// One discovered HiGoOS device, built from a UDP "announce" datagram
// (protocol HIGOOS/1, see docs/desktop-scanner.md §4).
struct Device {
    QString deviceId;       // stable id, e.g. "hg-7a3f9c12"
    QString model;          // "HiGoOS NAS"
    QString version;        // "1.4.0"
    QString hostname;       // "higoos"
    bool initialized = false; // false -> first-boot setup, true -> login
    int httpPort = 8080;
    bool https = false;
    QString primaryMac;
    QStringList addrs;      // reachable IPv4 addresses
    QString netMode;        // "dhcp" | "static"
    qint64 uptimeSec = 0;

    QString sourceAddr;     // address the datagram actually came from
    QDateTime lastSeen;     // updated every time we hear from it

    // Best address to reach the web/REST: the source address is what
    // actually answered us, so prefer it; fall back to the first advertised.
    QString primaryAddr() const {
        if (!sourceAddr.isEmpty()) return sourceAddr;
        return addrs.isEmpty() ? QString() : addrs.first();
    }

    QString webUrl() const {
        const QString scheme = https ? QStringLiteral("https") : QStringLiteral("http");
        return QStringLiteral("%1://%2:%3/").arg(scheme, primaryAddr()).arg(httpPort);
    }

    QString apiBase() const {
        const QString scheme = https ? QStringLiteral("https") : QStringLiteral("http");
        return QStringLiteral("%1://%2:%3").arg(scheme, primaryAddr()).arg(httpPort);
    }
};
