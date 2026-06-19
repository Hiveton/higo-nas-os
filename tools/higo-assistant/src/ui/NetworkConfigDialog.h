#pragma once

#include <QDialog>

#include "discovery/Device.h"

class QComboBox;
class QLabel;
class QLineEdit;
class QPushButton;
class HigoClient;

// First-boot network setup for one device. Flow:
//   1. log in (admin) to obtain a session
//   2. PUT /network/config -> review the governance impact summary
//   3. POST /network/config/confirm -> apply
// IP changes are high risk, hence the two-phase confirm. See
// docs/desktop-scanner.md §5.3 and docs/security-governance.md.
class NetworkConfigDialog : public QDialog {
    Q_OBJECT
public:
    explicit NetworkConfigDialog(const Device &device, QWidget *parent = nullptr);

private slots:
    void onModeChanged();
    void doLogin();
    void doApply(); // plan -> confirm

private:
    QJsonObject collectConfig() const;
    void setBusy(bool busy);
    void setStatus(const QString &text, bool error = false);

    Device m_device;
    HigoClient *m_client = nullptr;
    bool m_authed = false;
    QString m_pendingConfirmationId;

    QLineEdit *m_user = nullptr;
    QLineEdit *m_pass = nullptr;
    QPushButton *m_loginBtn = nullptr;

    QComboBox *m_mode = nullptr; // dhcp | static
    QLineEdit *m_hostname = nullptr;
    QLineEdit *m_address = nullptr;
    QLineEdit *m_prefix = nullptr;
    QLineEdit *m_gateway = nullptr;
    QLineEdit *m_dns = nullptr;

    QPushButton *m_applyBtn = nullptr;
    QLabel *m_status = nullptr;
};
