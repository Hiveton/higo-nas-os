#include <QApplication>

#include "ui/MainWindow.h"

int main(int argc, char *argv[]) {
    QApplication app(argc, argv);
    app.setApplicationName(QStringLiteral("HiGoOS 助手"));
    app.setOrganizationName(QStringLiteral("HiGoOS"));

    MainWindow w;
    w.show();
    return app.exec();
}
