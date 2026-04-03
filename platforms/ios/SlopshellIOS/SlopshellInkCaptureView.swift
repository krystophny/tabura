import PencilKit
import SwiftUI

struct SlopshellInkCaptureView: UIViewRepresentable {
    private static let penWidth = 2.4
    let onCommit: ([SlopshellInkStroke]) -> Void

    func makeCoordinator() -> Coordinator {
        Coordinator(onCommit: onCommit)
    }

    func makeUIView(context: Context) -> PKCanvasView {
        let canvas = PKCanvasView()
        canvas.backgroundColor = .clear
        canvas.isOpaque = false
        canvas.drawingPolicy = .pencilOnly
        canvas.delegate = context.coordinator
        canvas.tool = PKInkingTool(.pen, color: .black, width: Self.penWidth)
        return canvas
    }

    func updateUIView(_ uiView: PKCanvasView, context: Context) {
        uiView.delegate = context.coordinator
    }

    final class Coordinator: NSObject, PKCanvasViewDelegate {
        private let onCommit: ([SlopshellInkStroke]) -> Void
        private var lastStrokeCount = 0

        init(onCommit: @escaping ([SlopshellInkStroke]) -> Void) {
            self.onCommit = onCommit
        }

        func canvasViewDrawingDidChange(_ canvasView: PKCanvasView) {
            let drawing = canvasView.drawing
            guard drawing.strokes.count > lastStrokeCount else {
                return
            }
            let strokeSlice = drawing.strokes[lastStrokeCount...]
            let newStrokes = strokeSlice.compactMap(makeStroke)
            lastStrokeCount = drawing.strokes.count
            guard !newStrokes.isEmpty else {
                return
            }
            onCommit(newStrokes)
        }

        private func makeStroke(from stroke: PKStroke) -> SlopshellInkStroke? {
            let points = stroke.path.map(makePoint)
            guard !points.isEmpty else {
                return nil
            }
            return SlopshellInkStroke(
                pointerType: "pencil",
                width: SlopshellInkCaptureView.penWidth,
                points: points
            )
        }

        private func makePoint(from point: PKStrokePoint) -> SlopshellInkPoint {
            SlopshellInkPoint(
                x: point.location.x,
                y: point.location.y,
                pressure: Double(point.force),
                tiltX: Double(point.azimuth),
                tiltY: Double(point.altitude),
                roll: 0,
                timestampMS: point.timeOffset * 1000
            )
        }
    }
}
