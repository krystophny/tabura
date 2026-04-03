// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "SloppadFlowContract",
    products: [
        .library(
            name: "SloppadFlowContract",
            targets: ["SloppadFlowContract"]
        ),
        .library(
            name: "SloppadIOSModels",
            targets: ["SloppadIOSModels"]
        ),
    ],
    targets: [
        .target(
            name: "SloppadFlowContract"
        ),
        .target(
            name: "SloppadIOSModels",
            path: "SloppadIOS",
            exclude: [
                "ContentView.swift",
                "Info.plist",
                "SloppadAppModel.swift",
                "SloppadAudioCapture.swift",
                "SloppadCanvasTransport.swift",
                "SloppadCanvasWebView.swift",
                "SloppadChatTransport.swift",
                "SloppadIOSApp.swift",
                "SloppadInkCaptureView.swift",
                "SloppadServerDiscovery.swift",
            ],
            sources: ["SloppadModels.swift"]
        ),
        .testTarget(
            name: "SloppadFlowContractTests",
            dependencies: ["SloppadFlowContract"],
            resources: [.process("Resources")]
        ),
        .testTarget(
            name: "SloppadIOSModelsTests",
            dependencies: ["SloppadIOSModels"],
            path: "Tests/SloppadIOSModelsTests"
        ),
    ]
)
