import type { AdminPlugin } from '@go-cms/admin/sdk'
import NoticePage from './NoticePage.vue'
import TextEditor from './TextEditor.vue'
import NoticeConfig from './NoticeConfig.vue'
export const noticePlugin: AdminPlugin = {
  code: 'example',
  routes: [{ name: 'example.notice', path: '/admin/example', component: NoticePage }],
  fieldEditors: { 'example.text': TextEditor },
  configEditors: { 'example.notice': NoticeConfig },
}
