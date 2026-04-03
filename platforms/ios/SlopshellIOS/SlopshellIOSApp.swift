import SwiftUI

@main
struct SlopshellIOSApp: App {
    var body: some Scene {
        WindowGroup {
            if ProcessInfo.processInfo.arguments.contains("-SlopshellFlowHarness") {
                SlopshellFlowHarnessRootView(
                    preconditions: parseSlopshellFlowHarnessPreconditions(
                        ProcessInfo.processInfo.environment["SLOPSHELL_FLOW_PRECONDITIONS_JSON"]
                    )
                )
            } else {
                ContentView()
            }
        }
    }
}
