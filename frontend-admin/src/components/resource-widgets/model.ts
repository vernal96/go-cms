import type { ResourceWidget, WidgetArea } from '../../types/admin'

export interface WidgetSettingsValue {
  view: string
  columns: number
  margin_top: number
  margin_bottom: number
  enabled: boolean
  params: Record<string, unknown>
}

const areas: WidgetArea[] = ['body', 'sidebar']

export function sortWidgets(source: ResourceWidget[]): ResourceWidget[] {
  return [...source].sort((left, right) => {
    const areaDifference = areas.indexOf(left.area) - areas.indexOf(right.area)
    return areaDifference || left.position - right.position || left.id - right.id
  })
}

export function normalizeWidgetPositions(source: ResourceWidget[]): ResourceWidget[] {
  const positions: Record<WidgetArea, number> = { body: 0, sidebar: 0 }
  return sortWidgets(source).map((widget) => ({
    ...widget,
    params: { ...widget.params },
    position: positions[widget.area]++,
  }))
}

export function moveWidget(
  source: ResourceWidget[],
  id: number,
  area: WidgetArea,
  targetIndex: number,
): ResourceWidget[] {
  const moving = source.find((widget) => widget.id === id)
  if (!moving) return normalizeWidgetPositions(source)
  const remaining = source.filter((widget) => widget.id !== id)
  const target = remaining
    .filter((widget) => widget.area === area)
    .sort((left, right) => left.position - right.position)
  const adjustedIndex = moving.area === area && moving.position < targetIndex
    ? targetIndex - 1
    : targetIndex
  target.splice(Math.max(0, Math.min(adjustedIndex, target.length)), 0, {
    ...moving,
    area,
  })
  const positionedTarget = target.map((widget, position) => ({ ...widget, position }))
  const otherArea = area === 'body' ? 'sidebar' : 'body'
  return normalizeWidgetPositions([
    ...positionedTarget,
    ...remaining.filter((widget) => widget.area === otherArea),
  ])
}

export function widgetOrder(source: ResourceWidget[]): Array<{
  id: number
  area: WidgetArea
  position: number
}> {
  return normalizeWidgetPositions(source).map(({ id, area, position }) => ({
    id,
    area,
    position,
  }))
}
