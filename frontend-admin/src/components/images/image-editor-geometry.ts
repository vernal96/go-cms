import type Cropper from 'cropperjs'

export interface Size { width: number; height: number }

export function imageSize(original: Size, data: Cropper.Data): Size {
  const width = original.width * Math.abs(data.scaleX)
  const height = original.height * Math.abs(data.scaleY)
  return Math.abs(data.rotate % 180) === 90 ? { width: height, height: width } : { width, height }
}

export function fitScale(image: Size, viewport: Size): number {
  return Math.min(viewport.width / image.width, viewport.height / image.height)
}

export function rotateSelection(data: Cropper.Data, image: Size, angle: 90 | -90): Cropper.Data {
  return {
    ...data,
    x: angle === 90 ? image.height - data.y - data.height : data.y,
    y: angle === 90 ? data.x : image.width - data.x - data.width,
    width: data.height,
    height: data.width,
    rotate: ((data.rotate + angle) % 360 + 360) % 360,
  }
}

export function editorView(data: Cropper.Data, image: Size, viewport: Size, zoom: number) {
  const scale = fitScale(image, viewport) * zoom
  const width = image.width * scale
  const height = image.height * scale
  // Cropper clips selections to its container. Give it enough room to retain
  // the entire selection even when manual zoom puts it outside the viewport.
  const stage = { width: Math.ceil(Math.max(viewport.width, width)), height: Math.ceil(Math.max(viewport.height, height)) }
  return {
    stage,
    canvas: {
      width,
      left: stage.width / 2 - (width > viewport.width ? (data.x + data.width / 2) * scale : width / 2),
      top: stage.height / 2 - (height > viewport.height ? (data.y + data.height / 2) * scale : height / 2),
    },
  }
}
