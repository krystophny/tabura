// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "SlopshellFlowContract",
    products: [
        .library(
            name: "SlopshellFlowContract",
            targets: ["SlopshellFlowContract"]
        ),
        .library(
            name: "SlopshellIOSModels",
            targets: ["SlopshellIOSModels"]
        ),
    ],
    targets: [
        .target(
            name: "SlopshellFlowContract"
        ),
        .target(
            name: "SlopshellIOSModels",
            path: "SlopshellIOS",
            exclude: [
                "ContentView.swift",
                "Info.plist",
                "SlopshellAppModel.swift",
                "SlopshellAudioCapture.swift",
                "SlopshellCanvasTransport.swift",
                "SlopshellCanvasWebView.swift",
                "SlopshellChatTransport.swift",
                "SlopshellIOSApp.swift",
                "SlopshellInkCaptureView.swift",
                "SlopshellServerDiscovery.swift",
            ],
            sources: ["SlopshellModels.swift"]
        ),
        .testTarget(
            name: "SlopshellFlowContractTests",
            dependencies: ["SlopshellFlowContract"],
            resources: [.process("Resources")]
        ),
        .testTarget(
            name: "SlopshellIOSModelsTests",
            dependencies: ["SlopshellIOSModels"],
            path: "Tests/SlopshellIOSModelsTests"
        ),
    ]
)
