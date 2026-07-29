import type { ResourceNode } from '../types/resource'

export function filterResourceTree(
  nodes: ResourceNode[],
  query: string,
): ResourceNode[] {
  const normalizedQuery = query.trim().toLocaleLowerCase('ru-RU')
  if (normalizedQuery === '') {
    return nodes
  }

  return nodes.reduce<ResourceNode[]>((result, node) => {
    const matches = node.title
      .toLocaleLowerCase('ru-RU')
      .includes(normalizedQuery)

    if (matches) {
      result.push(node)
      return result
    }

    const children = filterResourceTree(node.children ?? [], normalizedQuery)
    if (children.length > 0) {
      result.push({
        ...node,
        children,
      })
    }

    return result
  }, [])
}
