import { describe, expect, it } from 'vitest'
import { editorView, fitScale, imageSize, rotateSelection } from './image-editor-geometry'

describe('image editor geometry', () => {
  for (const original of [{ width: 1200, height: 800 }, { width: 800, height: 1200 }, { width: 800, height: 800 }]) {
    for (const viewport of [{ width: 740, height: 520 }, { width: 320, height: 440 }]) {
      for (const scaleX of [1, -1]) {
        it(`preserves selection, fit and zoom for ${JSON.stringify({ original, viewport, scaleX })}`, () => {
          const data = { x: 31.25, y: 47.5, width: 500.5, height: 600.25, rotate: 0, scaleX, scaleY: -1 }
          for (const zoom of [1, 1.5, 3]) {
            let rotated = data
            for (let turn = 0; turn < 4; turn++) {
              const before = rotated
              rotated = rotateSelection(before, imageSize(original, before), 90)
              expect(rotateSelection(rotated, imageSize(original, rotated), -90)).toEqual(before)
              const size = imageSize(original, rotated)
              const view = editorView(rotated, size, viewport, zoom)
              expect(view.canvas.width / size.width / fitScale(size, viewport)).toBeCloseTo(zoom)
              if (zoom === 1) {
                expect(view.canvas.width).toBeLessThanOrEqual(viewport.width + 1e-9)
                expect(size.height * fitScale(size, viewport)).toBeLessThanOrEqual(viewport.height + 1e-9)
              }
              const scale = view.canvas.width / size.width
              expect(view.canvas.left + rotated.x * scale).toBeGreaterThanOrEqual(-1e-9)
              expect(view.canvas.top + rotated.y * scale).toBeGreaterThanOrEqual(-1e-9)
              expect(view.canvas.left + (rotated.x + rotated.width) * scale).toBeLessThanOrEqual(view.stage.width + 1e-9)
              expect(view.canvas.top + (rotated.y + rotated.height) * scale).toBeLessThanOrEqual(view.stage.height + 1e-9)
            }
            expect(rotated).toEqual(data)
          }
        })
      }
    }
  }
})
