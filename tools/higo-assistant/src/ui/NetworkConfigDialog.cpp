#include "ui/NetworkConfigDialog.h"

#include <QComboBox>
#include <QFormLayout>
#include <QGroupBox>
#include <QJsonArray>
#include <QJsonObject>
#include <QLabel>
#include <QLineEdit>
#include <QMessageBox>
#include <QPushButton>
#include <QVBoxLayout>

#include "api/HigoClient.h"

NetworkConfigDialog::NetworkConfigDialog(const Device &device, QWidget *parent)
    : QDialog(parent), m_device(device) {
    setWindowTitle(tr("配置网络 — %1").arg(device.hostname.isEmpty() ? device.deviceId : device.hostname));
    setModal(true);
    resize(440, 520);

    m_client = new HigoClient(this);
    m_client->setApiBase(device.apiBase());

    auto *root = new QVBoxLayout(this);

    // --- Login group (prod requires an admin session for network writes) ---
    auto *authBox = new QGroupBox(tr("管理员登录"), this);
    auto *authForm = new QFormLayout(authBox);
    m_user = new QLineEdit(authBox);
    m_user->setText(QStringLiteral("admin"));
    m_pass = new QLineEdit(authBox);
    m_pass->setEchoMode(QLineEdit::Password);
    m_pass->setPlaceholderText(tr("首次开机的一次性管理员密码"));
    m_loginBtn = new QPushButton(tr("登录"), authBox);
    authForm->addRow(tr("账号"), m_user);
    authForm->addRow(tr("密码"), m_pass);
    authForm->addRow(QString(), m_loginBtn);
    root->addWidget(authBox);

    // --- Network config group ---
    auto *netBox = new QGroupBox(tr("网络设置"), this);
    auto *netForm = new QFormLayout(netBox);
    m_mode = new QComboBox(netBox);
    m_mode->addItem(tr("自动获取 (DHCP)"), QStringLiteral("dhcp"));
    m_mode->addItem(tr("手动 (静态 IP)"), QStringLiteral("static"));
    m_hostname = new QLineEdit(device.hostname, netBox);
    m_address = new QLineEdit(device.primaryAddr(), netBox);
    m_prefix = new QLineEdit(QStringLiteral("24"), netBox);
    m_gateway = new QLineEdit(netBox);
    m_dns = new QLineEdit(QStringLiteral("223.5.5.5, 8.8.8.8"), netBox);

    netForm->addRow(tr("地址分配"), m_mode);
    netForm->addRow(tr("主机名"), m_hostname);
    netForm->addRow(tr("IP 地址"), m_address);
    netForm->addRow(tr("前缀长度"), m_prefix);
    netForm->addRow(tr("网关"), m_gateway);
    netForm->addRow(tr("DNS(逗号分隔)"), m_dns);
    root->addWidget(netBox);

    m_applyBtn = new QPushButton(tr("应用"), this);
    m_applyBtn->setEnabled(false);
    root->addWidget(m_applyBtn);

    m_status = new QLabel(this);
    m_status->setWordWrap(true);
    root->addWidget(m_status);

    connect(m_mode, &QComboBox::currentIndexChanged, this, &NetworkConfigDialog::onModeChanged);
    connect(m_loginBtn, &QPushButton::clicked, this, &NetworkConfigDialog::doLogin);
    connect(m_applyBtn, &QPushButton::clicked, this, &NetworkConfigDialog::doApply);

    // Prefill mode from the device fingerprint.
    const int idx = m_mode->findData(device.netMode.isEmpty() ? QStringLiteral("dhcp") : device.netMode);
    if (idx >= 0) m_mode->setCurrentIndex(idx);
    onModeChanged();
}

void NetworkConfigDialog::onModeChanged() {
    const bool isStatic = m_mode->currentData().toString() == QStringLiteral("static");
    m_address->setEnabled(isStatic);
    m_prefix->setEnabled(isStatic);
    m_gateway->setEnabled(isStatic);
    m_dns->setEnabled(isStatic);
}

void NetworkConfigDialog::setBusy(bool busy) {
    m_loginBtn->setEnabled(!busy);
    m_applyBtn->setEnabled(!busy && m_authed);
}

void NetworkConfigDialog::setStatus(const QString &text, bool error) {
    m_status->setText(text);
    m_status->setStyleSheet(error ? QStringLiteral("color:#c0392b;")
                                  : QStringLiteral("color:#27ae60;"));
}

void NetworkConfigDialog::doLogin() {
    if (m_pass->text().isEmpty()) {
        setStatus(tr("请输入管理员密码。"), true);
        return;
    }
    setBusy(true);
    setStatus(tr("正在登录……"));
    m_client->login(m_user->text(), m_pass->text(),
                    [this](bool ok, const QJsonObject &, const QString &err) {
                        setBusy(false);
                        if (!ok) {
                            setStatus(tr("登录失败:%1").arg(err), true);
                            return;
                        }
                        m_authed = true;
                        m_applyBtn->setEnabled(true);
                        setStatus(tr("登录成功,可以应用网络设置。"));
                    });
}

QJsonObject NetworkConfigDialog::collectConfig() const {
    QJsonObject cfg;
    cfg[QStringLiteral("mode")] = m_mode->currentData().toString();
    cfg[QStringLiteral("hostname")] = m_hostname->text().trimmed();
    if (m_mode->currentData().toString() == QStringLiteral("static")) {
        cfg[QStringLiteral("address")] = m_address->text().trimmed();
        cfg[QStringLiteral("prefix")] = m_prefix->text().trimmed().toInt();
        cfg[QStringLiteral("gateway")] = m_gateway->text().trimmed();
        QJsonArray dns;
        const auto parts = m_dns->text().split(QLatin1Char(','), Qt::SkipEmptyParts);
        for (const QString &p : parts) dns.append(p.trimmed());
        cfg[QStringLiteral("dns")] = dns;
    }
    return cfg;
}

void NetworkConfigDialog::doApply() {
    if (!m_authed) {
        setStatus(tr("请先登录。"), true);
        return;
    }
    setBusy(true);
    setStatus(tr("正在校验变更……"));

    // Phase 1: plan -> impact summary + confirmationId.
    m_client->planNetworkConfig(collectConfig(),
        [this](bool ok, const QJsonObject &data, const QString &err) {
            if (!ok) {
                setBusy(false);
                setStatus(tr("校验失败:%1").arg(err), true);
                return;
            }
            m_pendingConfirmationId = data.value(QStringLiteral("confirmationId")).toString();
            const QString impact = data.value(QStringLiteral("impactSummary")).toString(
                tr("修改 IP 可能会中断当前连接,设备地址变更后需要重新搜索。"));

            const auto choice = QMessageBox::warning(
                this, tr("确认应用网络变更"), impact,
                QMessageBox::Ok | QMessageBox::Cancel, QMessageBox::Cancel);
            if (choice != QMessageBox::Ok || m_pendingConfirmationId.isEmpty()) {
                setBusy(false);
                setStatus(tr("已取消。"));
                return;
            }

            // Phase 2: confirm -> apply.
            setStatus(tr("正在应用……"));
            m_client->confirmNetworkConfig(m_pendingConfirmationId,
                [this](bool ok2, const QJsonObject &, const QString &err2) {
                    setBusy(false);
                    if (!ok2) {
                        setStatus(tr("应用失败:%1").arg(err2), true);
                        return;
                    }
                    QMessageBox::information(
                        this, tr("已应用"),
                        tr("网络设置已下发。设备 IP 可能已变化,关闭后请重新搜索设备。"));
                    accept();
                });
        });
}
