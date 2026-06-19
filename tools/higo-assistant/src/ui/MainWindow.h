#pragma once

#include <QHash>
#include <QMainWindow>

#include "discovery/Device.h"

class QLabel;
class QPushButton;
class QTableWidget;
class UdpDiscoverer;

// Main window: live list of discovered HiGoOS devices. The actual product
// features live in the device's web UI ("Open Web"); this tool only owns the
// pre-web steps a browser can't do — discovery and first-boot IP setup.
class MainWindow : public QMainWindow {
    Q_OBJECT
public:
    explicit MainWindow(QWidget *parent = nullptr);

private slots:
    void onDeviceDiscovered(const Device &device);
    void onDeviceLost(const QString &deviceId);
    void onSelectionChanged();
    void openWeb();
    void configureNetwork();

private:
    void rebuildRow(const Device &device);
    Device selectedDevice() const;

    UdpDiscoverer *m_discoverer = nullptr;
    QTableWidget *m_table = nullptr;
    QPushButton *m_openWebBtn = nullptr;
    QPushButton *m_configBtn = nullptr;
    QLabel *m_status = nullptr;

    QHash<QString, Device> m_devices; // deviceId -> latest
    QHash<QString, int> m_rowOf;      // deviceId -> table row
};
