#pragma once

#include <QHash>
#include <QObject>
#include <QTimer>

#include "discovery/Device.h"

class QUdpSocket;

// Broadcasts HIGOOS/1 "discover" datagrams across every active interface and
// collects "announce" replies. Devices not heard from within a staleness
// window are reported as lost. See docs/desktop-scanner.md §4.
class UdpDiscoverer : public QObject {
    Q_OBJECT
public:
    explicit UdpDiscoverer(QObject *parent = nullptr);

    // Begin periodic scanning. Idempotent.
    void start();
    void stop();

    // Fire a single discovery round immediately (e.g. on a Refresh button).
    void scanOnce();

signals:
    void deviceDiscovered(const Device &device); // new or updated
    void deviceLost(const QString &deviceId);

private slots:
    void readPending();
    void pruneStale();

private:
    void sendDiscover();
    static QByteArray buildDiscover(const QString &nonce);

    QUdpSocket *m_socket = nullptr;
    QTimer m_scanTimer;
    QTimer m_pruneTimer;
    QString m_nonce;
    QHash<QString, Device> m_devices; // deviceId -> latest

    static constexpr quint16 kDiscoveryPort = 19999;
    static constexpr int kScanIntervalMs = 3000;
    static constexpr int kStaleAfterMs = 12000;
};
