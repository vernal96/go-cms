export type ResourceKind = 'page' | 'section' | 'link'

export interface ResourceNode {
  id: string
  title: string
  type: ResourceKind
  children?: ResourceNode[]
}
