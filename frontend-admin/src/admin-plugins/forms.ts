import MailActionEditor from './forms/MailActionEditor.vue'
import FormPickerField from './forms/FormPickerField.vue'
import ResultsPageSizeField from './forms/ResultsPageSizeField.vue'
import { Tickets } from '@element-plus/icons-vue'
import type { AdminPlugin } from './plugin'
import FormBuilderView from './forms/FormBuilderView.vue'
import FormResultDetailView from './forms/FormResultDetailView.vue'
import FormsListView from './forms/FormsListView.vue'
import FormsResultsView from './forms/FormsResultsView.vue'

export const formsAdminPlugin: AdminPlugin = {
  code: 'forms',
  configEditors: { 'forms.mail': MailActionEditor },
  fieldEditors: { 'forms.form-picker': FormPickerField, 'forms.results-page-size': ResultsPageSizeField },
  icons: { forms: Tickets },
  routes: [
    { name: 'forms.list', path: '/admin/forms', component: FormsListView },
    { name: 'forms.edit', path: '/admin/forms/:formId', component: FormBuilderView },
    { name: 'forms.results', path: '/admin/forms-results', component: FormsResultsView },
    { name: 'forms.results.detail', path: '/admin/forms-results/:resultId', component: FormResultDetailView },
  ],
}
