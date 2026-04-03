import SwiftUI

@main
struct SloppadIOSApp: App {
    var body: some Scene {
        WindowGroup {
            if ProcessInfo.processInfo.arguments.contains("-SloppadFlowHarness") {
                SloppadFlowHarnessRootView(
                    preconditions: parseSloppadFlowHarnessPreconditions(
                        ProcessInfo.processInfo.environment["SLOPPAD_FLOW_PRECONDITIONS_JSON"]
                    )
                )
            } else {
                ContentView()
            }
        }
    }
}
