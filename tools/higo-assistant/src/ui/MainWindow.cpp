#include "ui/MainWindow.h"

#include <QDesktopServices>
#include <QHBoxLayout>
#include <QHeaderView>
#include <QLabel>
#include <QMessageBox>
#include <QPushButton>
#include <QTableWidget>
#include <QUrl>
#include <QVBoxLayout>
#include <QWidget>

#include "discovery/UdpDiscoverer.h"
#include "ui/NetworkConfigDialog.h"

namespace {
enum Column { ColModel = 0, ColHostname, ColAddress, ColVersion, ColStatus, ColCount };
}

MainWindow::MainWindow(QWidget *parent) : QMainWindow(parent) {
    setWindowTitle(tr("HiGoOS 助手"));
    resize(820, 480);

    auto *central = new QWidget(this);
    auto *root = new QVBoxLayout(central);

    auto *hint = new QLabel(
        tr("正在局域网内搜索 HiGoOS 设备…… 选中设备后可「打开管理界面」或「配置网络」。"), central);
    hint->setWordWrap(true);
    root->addWidget(hint);

    m_table = new QTableWidget(0, ColCount, central);
    m_table->setHorizontalHeaderLabels(
        {tr("型号"), tr("主机名"), tr("IP 地址"), tr("版本"), tr("状态")});
    m_table->horizontalHeader()->setStretchLastSection(true);
    m_table->setSelectionBehavior(QAbstractItemView::SelectRows);
    m_table->setSelectionMode(QAbstractItemView::SingleSelection);
    m_table->setEditTriggers(QAbstractItemView::NoEditTriggers);
    m_table->verticalHeader()->setVisible(false);
    root->addWidget(m_table, 1);

    auto *buttons = new QHBoxLayout();
    auto *refreshBtn = new QPushButton(tr("刷新"), central);
    m_openWebBtn = new QPushButton(tr("打开管理界面"), central);
    m_configBtn = new QPushButton(tr("配置网络"), central);
    m_openWebBtn->setEnabled(false);
    m_configBtn->setEnabled(false);
    buttons->addWidget(refreshBtn);
    buttons->addStretch(1);
    buttons->addWidget(m_configBtn);
    buttons->addWidget(m_openWebBtn);
    root->addLayout(buttons);

    m_status = new QLabel(tr("已发现 0 台设备"), central);
    root->addWidget(m_status);

    setCentralWidget(central);

    m_discoverer = new UdpDiscoverer(this);
    connect(m_discoverer, &UdpDiscoverer::deviceDiscovered, this, &MainWindow::onDeviceDiscovered);
    connect(m_discoverer, &UdpDiscoverer::deviceLost, this, &MainWindow::onDeviceLost);
    connect(refreshBtn, &QPushButton::clicked, m_discoverer, &UdpDiscoverer::scanOnce);
    connect(m_table, &QTableWidget::itemSelectionChanged, this, &MainWindow::onSelectionChanged);
    connect(m_openWebBtn, &QPushButton::clicked, this, &MainWindow::openWeb);
    connect(m_configBtn, &QPushButton::clicked, this, &MainWindow::configureNetwork);

    m_discoverer->start();
}

void MainWindow::onDeviceDiscovered(const Device &device) {
    m_devices.insert(device.deviceId, device);
    rebuildRow(device);
    m_status->setText(tr("已发现 %1 台设备").arg(m_devices.size()));
}

void MainWindow::onDeviceLost(const QString &deviceId) {
    m_devices.remove(deviceId);
    if (m_rowOf.contains(deviceId)) {
        const int row = m_rowOf.take(deviceId);
        m_table->removeRow(row);
        // Rows below shifted up; rebuild the index map.
        for (auto it = m_rowOf.begin(); it != m_rowOf.end(); ++it) {
            if (it.value() > row) it.value() -= 1;
        }
    }
    m_status->setText(tr("已发现 %1 台设备").arg(m_devices.size()));
    onSelectionChanged();
}

void MainWindow::rebuildRow(const Device &d) {
    int row;
    if (m_rowOf.contains(d.deviceId)) {
        row = m_rowOf.value(d.deviceId);
    } else {
        row = m_table->rowCount();
        m_table->insertRow(row);
        m_rowOf.insert(d.deviceId, row);
        for (int c = 0; c < ColCount; ++c) m_table->setItem(row, c, new QTableWidgetItem());
        m_table->item(row, ColModel)->setData(Qt::UserRole, d.deviceId);
    }
    const QString status = d.initialized ? tr("已初始化") : tr("待初始化");
    m_table->item(row, ColModel)->setText(d.model.isEmpty() ? tr("HiGoOS") : d.model);
    m_table->item(row, ColHostname)->setText(d.hostname);
    m_table->item(row, ColAddress)->setText(d.primaryAddr());
    m_table->item(row, ColVersion)->setText(d.version);
    m_table->item(row, ColStatus)->setText(status);
}

Device MainWindow::selectedDevice() const {
    const auto rows = m_table->selectionModel() ? m_table->selectionModel()->selectedRows() : QModelIndexList();
    if (rows.isEmpty()) return {};
    const QTableWidgetItem *item = m_table->item(rows.first().row(), ColModel);
    if (!item) return {};
    const QString id = item->data(Qt::UserRole).toString();
    return m_devices.value(id);
}

void MainWindow::onSelectionChanged() {
    const bool has = !selectedDevice().deviceId.isEmpty();
    m_openWebBtn->setEnabled(has);
    m_configBtn->setEnabled(has);
}

void MainWindow::openWeb() {
    const Device d = selectedDevice();
    if (d.deviceId.isEmpty()) return;
    if (d.primaryAddr().isEmpty()) {
        QMessageBox::warning(this, tr("无法打开"),
                             tr("该设备暂无可达 IP,请先为它配置网络。"));
        return;
    }
    // All real product features live in the device's web UI.
    QDesktopServices::openUrl(QUrl(d.webUrl()));
}

void MainWindow::configureNetwork() {
    const Device d = selectedDevice();
    if (d.deviceId.isEmpty()) return;
    if (d.primaryAddr().isEmpty()) {
        QMessageBox::warning(this, tr("无法配置"),
                             tr("该设备当前不可达。若它与本机不在同一网段,请参考文档处理。"));
        return;
    }
    NetworkConfigDialog dlg(d, this);
    dlg.exec();
}
