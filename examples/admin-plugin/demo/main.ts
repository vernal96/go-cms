import { createApp, defineComponent, h, onMounted, ref } from 'vue'
import { createRouter, createWebHistory, RouterView } from 'vue-router'
import { noticePlugin } from '../dist/notice.js'
import { AdminPluginRegistry, adminPluginRegistryKey, adminAccessTokenKey, adminPermissionsKey, useSelectedSite, adminRequest } from '@go-cms/admin/sdk'
import 'element-plus/dist/index.css'
import '@go-cms/admin/sdk.css'
const missing=new URLSearchParams(location.search).has('missing')
const registry=new AdminPluginRegistry([{...noticePlugin,...(missing?{fieldEditors:{}}:{})}])
const token=ref(sessionStorage.getItem('example-token')??'')
const selected=useSelectedSite()
const sites=ref<Array<{id:number;domain:string}>>([])
const router=createRouter({history:createWebHistory(),routes:[{path:'/',redirect:'/admin/example'},...registry.routeRecords()]})
const shell=defineComponent({setup(){
 onMounted(async()=>{sites.value=(await adminRequest<{items:typeof sites.value}>('/api/sites',token.value)).items;selected.setSelected(sites.value[0]??null)})
 return ()=>h('div',[
  h('label',['Сайт ',h('select',{'aria-label':'Сайт',value:selected.selectedSite.value?.id,onChange:(event:Event)=>selected.setSelected(sites.value.find(site=>site.id===Number((event.target as HTMLSelectElement).value))??null)},sites.value.map(site=>h('option',{value:site.id},site.domain)))]),
  h(RouterView),
 ])
}})
createApp(shell).provide(adminPluginRegistryKey,registry).provide(adminAccessTokenKey,token).provide(adminPermissionsKey,ref(new Set<string>())).use(router).mount('#app')
