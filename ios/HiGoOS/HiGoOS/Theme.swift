import SwiftUI

enum HiGoTheme {
    static let blue = Color(red: 0.07, green: 0.47, blue: 1.0)
    static let cyan = Color(red: 0.09, green: 0.78, blue: 0.86)
    static let green = Color(red: 0.13, green: 0.62, blue: 0.31)
    static let orange = Color(red: 0.95, green: 0.55, blue: 0.08)
    static let red = Color(red: 0.93, green: 0.18, blue: 0.22)
    static let ink = Color(red: 0.06, green: 0.11, blue: 0.18)
    static let muted = Color(red: 0.42, green: 0.49, blue: 0.58)
    static let page = Color(red: 0.96, green: 0.98, blue: 1.0)
    static let card = Color.white

    static let corner: CGFloat = 18
    static let compactCorner: CGFloat = 14
    static let shadow = Color(red: 0.15, green: 0.31, blue: 0.50).opacity(0.10)
}

extension Color {
    static var higoBlue: Color { HiGoTheme.blue }
    static var higoGreen: Color { HiGoTheme.green }
    static var higoOrange: Color { HiGoTheme.orange }
    static var higoRed: Color { HiGoTheme.red }
}
