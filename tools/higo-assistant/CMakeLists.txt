cmake_minimum_required(VERSION 3.21)

project(HiGoAssistant
    VERSION 0.1.0
    DESCRIPTION "HiGoOS LAN scanner & first-boot network setup tool"
    LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTOUIC ON)
set(CMAKE_AUTORCC ON)

# Qt 6 (Widgets desktop app). Set CMAKE_PREFIX_PATH to your Qt install if not found,
# e.g. -DCMAKE_PREFIX_PATH=~/Qt/6.7.0/macos
find_package(Qt6 REQUIRED COMPONENTS Widgets Network)

qt_standard_project_setup()

qt_add_executable(higo-assistant
    src/main.cpp
    src/discovery/Device.h
    src/discovery/UdpDiscoverer.h
    src/discovery/UdpDiscoverer.cpp
    src/api/HigoClient.h
    src/api/HigoClient.cpp
    src/ui/MainWindow.h
    src/ui/MainWindow.cpp
    src/ui/NetworkConfigDialog.h
    src/ui/NetworkConfigDialog.cpp
)

target_include_directories(higo-assistant PRIVATE src)

target_link_libraries(higo-assistant PRIVATE
    Qt6::Widgets
    Qt6::Network
)

# Place the binary in a predictable spot for local runs.
set_target_properties(higo-assistant PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)

# Convenience: macOS/Windows bundle metadata can be added later via
# set_target_properties(... MACOSX_BUNDLE ON WIN32_EXECUTABLE ON).
