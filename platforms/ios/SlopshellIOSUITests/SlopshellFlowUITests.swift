import XCTest

final class SlopshellFlowUITests: XCTestCase {
    private struct FlowFixtureBundle: Decodable {
        let platform: String
        let flows: [FlowDefinition]
        let selectors: [String: String]
    }

    private struct FlowDefinition: Decodable {
        let name: String
        let preconditions: FlowPreconditions?
        let steps: [FlowStep]
    }

    private struct FlowPreconditions: Decodable {
        let tool: String?
        let session: String?
        let silent: Bool?
        let indicatorState: String?

        private enum CodingKeys: String, CodingKey {
            case tool
            case session
            case silent
            case indicatorState = "indicator_state"
        }

        func jsonString() -> String {
            var parts: [String] = []
            if let tool {
                parts.append("\"tool\":\"\(tool)\"")
            }
            if let session {
                parts.append("\"session\":\"\(session)\"")
            }
            if let silent {
                parts.append("\"silent\":\(silent ? "true" : "false")")
            }
            if let indicatorState, indicatorState.isEmpty == false {
                parts.append("\"indicator_state\":\"\(indicatorState)\"")
            }
            return "{\(parts.joined(separator: ","))}"
        }
    }

    private struct FlowStep: Decodable {
        let action: String
        let target: String?
        let durationMS: Int?
        let platforms: [String]?
        let expect: FlowExpectations?

        private enum CodingKeys: String, CodingKey {
            case action
            case target
            case durationMS = "duration_ms"
            case platforms
            case expect
        }
    }

    private struct FlowExpectations: Decodable {
        let activeTool: String?
        let session: String?
        let silent: Bool?
        let slopshellCircle: String?
        let dotInnerIcon: String?
        let bodyClassContains: String?
        let indicatorState: String?
        let cursorClass: String?

        private enum CodingKeys: String, CodingKey {
            case activeTool = "active_tool"
            case session
            case silent
            case slopshellCircle = "slopshell_circle"
            case dotInnerIcon = "dot_inner_icon"
            case bodyClassContains = "body_class_contains"
            case indicatorState = "indicator_state"
            case cursorClass = "cursor_class"
        }
    }

    func testSharedFlowsExecuteOnIOSHarness() throws {
        let bundle = try loadFixtureBundle()
        XCTAssertEqual(bundle.platform, "ios")
        XCTAssertFalse(bundle.flows.isEmpty)

        for flow in bundle.flows {
            let app = XCUIApplication()
            app.launchArguments = ["-SlopshellFlowHarness"]
            app.launchEnvironment["SLOPSHELL_FLOW_PRECONDITIONS_JSON"] = flow.preconditions?.jsonString() ?? "{}"
            app.launch()
            XCTAssertTrue(element("slopshell_circle_dot", in: app).waitForExistence(timeout: 5), flow.name)
            run(flow: flow, app: app, selectors: bundle.selectors)
            app.terminate()
            print("ios-ui PASS \(flow.name)")
        }
    }

    private func run(flow: FlowDefinition, app: XCUIApplication, selectors: [String: String]) {
        for step in flow.steps {
            if let platforms = step.platforms, platforms.contains("ios") == false {
                continue
            }
            switch step.action {
            case "tap":
                guard let target = step.target else {
                    XCTFail("missing tap target for \(flow.name)")
                    continue
                }
                element(selector(for: target, selectors: selectors), in: app).tap()
            case "tap_outside":
                element(selector(for: "canvas_viewport", selectors: selectors), in: app).tap()
            case "verify":
                if let target = step.target {
                    XCTAssertTrue(element(selector(for: target, selectors: selectors), in: app).exists, flow.name)
                }
            case "wait":
                usleep(useconds_t((step.durationMS ?? 0) * 1000))
            default:
                XCTFail("unsupported action \(step.action)")
            }
            assert(expectations: step.expect, app: app)
        }
    }

    private func assert(expectations: FlowExpectations?, app: XCUIApplication) {
        guard let expectations else {
            return
        }
        if let activeTool = expectations.activeTool {
            XCTAssertEqual(stateValue("flow_state_active_tool", app: app), activeTool)
        }
        if let session = expectations.session {
            XCTAssertEqual(stateValue("flow_state_session", app: app), session)
        }
        if let silent = expectations.silent {
            XCTAssertEqual(stateValue("flow_state_silent", app: app), silent ? "true" : "false")
        }
        if let slopshellCircle = expectations.slopshellCircle {
            XCTAssertEqual(stateValue("flow_state_slopshell_circle", app: app), slopshellCircle)
        }
        if let dotInnerIcon = expectations.dotInnerIcon {
            XCTAssertEqual(stateValue("flow_state_dot_inner_icon", app: app), dotInnerIcon)
        }
        if let indicatorState = expectations.indicatorState {
            XCTAssertEqual(stateValue("flow_state_indicator_state", app: app), indicatorState)
        }
        if let bodyClassContains = expectations.bodyClassContains {
            XCTAssertTrue(stateValue("flow_state_body_class", app: app).contains(bodyClassContains))
        }
        if let cursorClass = expectations.cursorClass {
            XCTAssertEqual(stateValue("flow_state_cursor_class", app: app), cursorClass)
        }
    }

    private func stateValue(_ id: String, app: XCUIApplication) -> String {
        element(id, in: app).label
    }

    private func selector(for logicalTarget: String, selectors: [String: String]) -> String {
        selectors[logicalTarget] ?? logicalTarget
    }

    private func element(_ identifier: String, in app: XCUIApplication) -> XCUIElement {
        app.descendants(matching: .any)[identifier]
    }

    private func loadFixtureBundle() throws -> FlowFixtureBundle {
        let bundle = Bundle(for: type(of: self))
        guard let url = bundle.url(forResource: "flow-fixtures", withExtension: "json") else {
            throw NSError(domain: "SlopshellFlowUITests", code: 1, userInfo: [NSLocalizedDescriptionKey: "missing flow-fixtures.json"])
        }
        let data = try Data(contentsOf: url)
        return try JSONDecoder().decode(FlowFixtureBundle.self, from: data)
    }
}
