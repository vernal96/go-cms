import { Message } from '@element-plus/icons-vue'
import type { AdminPlugin } from './plugin'
import MailHistoryView from './mail/MailHistoryView.vue'
import MailMessageDetailView from './mail/MailMessageDetailView.vue'
import MailSendView from './mail/MailSendView.vue'
import MailTemplateFormView from './mail/MailTemplateFormView.vue'
import MailTemplatesView from './mail/MailTemplatesView.vue'

export const mailAdminPlugin: AdminPlugin = {
  code: 'mail',
  icons: { mail: Message },
  routes: [
    { name: 'mail.templates', path: '/admin/mail/templates', component: MailTemplatesView },
    { name: 'mail.templates.create', path: '/admin/mail/templates/new', component: MailTemplateFormView },
    { name: 'mail.templates.edit', path: '/admin/mail/templates/:templateId', component: MailTemplateFormView },
    { name: 'mail.send', path: '/admin/mail/send', component: MailSendView },
    { name: 'mail.history', path: '/admin/mail/history', component: MailHistoryView },
    { name: 'mail.history.detail', path: '/admin/mail/history/:messageId', component: MailMessageDetailView },
  ],
}
