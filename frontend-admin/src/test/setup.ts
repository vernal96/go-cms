import { config } from '@vue/test-utils'

config.global.directives = {
  ...config.global.directives,
  loading: () => undefined,
}
