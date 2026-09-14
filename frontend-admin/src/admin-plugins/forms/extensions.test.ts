// @vitest-environment jsdom
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { AdminPluginRegistry } from '../registry'
import { adminPluginRegistryKey } from '../context'
import type { FormAction, FormField } from './types'
import FormFieldEditor from './FormFieldEditor.vue'
import FormElementEditor from './FormElementEditor.vue'
import FormActionDialog from './FormActionDialog.vue'

describe('contributed configuration editors', () => {
 it('preserves the backend int field step after opening and saving', async () => {
  const field:FormField={id:1,form_id:1,code:'count',type:'int',label:'Count',required:false,rules:[],options:{step:2},result_label:'',show_on_site:false,show_in_results:false,result_position:0,created_at:'',updated_at:''}
  const wrapper=mount(FormFieldEditor,{props:{disabled:false,initialType:'int',field,fields:[],availableTypes:[{code:'int',label:'Целое число',editor:'int',options:[{key:'step',label:'Шаг',type:'int',required:false}]}]}})
  await flushPromises()
  expect(wrapper.findComponent({name:'NumberField'}).exists()).toBe(true)
  expect((wrapper.vm as unknown as {payload():FormField}).payload().options).toEqual({step:2})
  wrapper.unmount()
 })
 it('preserves contributed option objects and renders their declared controls',async () => {
  const field:FormField={id:1,form_id:1,code:'custom',type:'example.custom',label:'Custom',required:false,rules:[],options:{limit:3,enabled:false,nested:{tags:['a']}},result_label:'',show_on_site:false,show_in_results:false,result_position:0,created_at:'',updated_at:''}
  const wrapper=mount(FormFieldEditor,{props:{disabled:false,initialType:field.type,field,fields:[],availableTypes:[{code:field.type,label:'Custom',options:[{key:'limit',label:'Limit',type:'int',required:false},{key:'enabled',label:'Enabled',type:'checkbox',required:false},{key:'nested',label:'Nested',type:'json',required:false}]}]}})
  await flushPromises()
  expect((wrapper.vm as unknown as {payload():FormField}).payload().options).toEqual(field.options)
  wrapper.unmount()
 })
 it('round-trips typed generic action settings',async () => {
  const config={count:3,enabled:false,nested:{tags:['a']}}
  const wrapper=mountAction(config)
  await flushPromises()
  await wrapper.findAllComponents({name:'ElButton'}).at(-1)!.trigger('click')
  expect(wrapper.emitted('save')?.[0]?.[0]).toMatchObject({config})
  wrapper.unmount()
 })
 it('resolves a contributed action editor by semantic code',async () => {
  const custom=defineComponent({props:['modelValue'],emits:['update:modelValue'],template:'<button class="custom-config" @click="$emit(\'update:modelValue\',{count:7,enabled:false})">Set config</button>'})
  const registry=new AdminPluginRegistry([{code:'example',configEditors:{'example.editor':custom}}])
  const wrapper=mountAction({},'example.editor',registry)
  await flushPromises()
  await wrapper.find('.custom-config').trigger('click')
  await wrapper.findAllComponents({name:'ElButton'}).at(-1)!.trigger('click')
  expect(wrapper.emitted('save')?.[0]?.[0]).toMatchObject({config:{count:7,enabled:false}})
  wrapper.unmount()
 })
 it('blocks save when the requested configuration editor is absent',async () => {
  const wrapper=mountAction({count:3},'missing.editor')
  await flushPromises()
  await wrapper.findAllComponents({name:'ElButton'}).at(-1)!.trigger('click')
  expect(wrapper.emitted('save')).toBeUndefined()
  expect(wrapper.text()).toContain('недоступен')
  wrapper.unmount()
 })
 it('edits a custom element through its metadata',async () => {
  const config={count:3,enabled:false}
  const wrapper=mount(FormElementEditor,{props:{disabled:false,initialType:'example.notice',element:{id:1,form_id:1,code:'notice',type:'example.notice',config,created_at:'',updated_at:''},availableTypes:[{code:'example.notice',label:'Notice',fields:[{key:'count',label:'Count',type:'int',required:true},{key:'enabled',label:'Enabled',type:'checkbox',required:false}]}],accessToken:'',permissions:new Set<string>()}})
  await flushPromises()
  expect((wrapper.vm as unknown as {payload():{config:unknown}}).payload().config).toEqual(config)
  expect(wrapper.findComponent({name:'NumberField'}).exists()).toBe(true)
  wrapper.unmount()
 })
 it('retains metadata defaults after initializing a new element',async () => {
  const wrapper=mount(FormElementEditor,{props:{disabled:false,initialType:'heading',availableTypes:[{code:'heading',label:'Heading',fields:[{key:'level',label:'Level',type:'int',required:true,default:2}]}],accessToken:'',permissions:new Set<string>()}})
  await flushPromises()
  expect((wrapper.vm as unknown as {payload():{config:unknown}}).payload().config).toEqual({level:2})
  wrapper.unmount()
 })
 it('blocks action save with invalid JSON instead of retaining the previous object',async () => {
  const wrapper=mountAction({nested:{tags:['a']}})
  await flushPromises()
  await wrapper.findComponent({name:'JsonField'}).find('textarea').setValue('{invalid')
  await wrapper.findAllComponents({name:'ElButton'}).at(-1)!.trigger('click')
  expect(wrapper.emitted('save')).toBeUndefined()
  expect(wrapper.text()).toContain('Некорректный JSON')
  wrapper.unmount()
 })
})

function mountAction(config:Record<string,unknown>,editor?:string,registry=new AdminPluginRegistry([])) {
 const action:FormAction={id:1,form_id:1,code:'custom',name:'Custom',enabled:true,trigger:{type:'submitted'},action_type:'example.action',config,position:0,created_at:'',updated_at:''}
 return mount(FormActionDialog,{props:{modelValue:true,action,actionTypes:[{code:action.action_type,label:'Custom',available:true,editor_code:editor,fields:[{key:'count',label:'Count',type:'int',required:false},{key:'enabled',label:'Enabled',type:'checkbox',required:false},{key:'nested',label:'Nested',type:'json',required:false}]}],fields:[],statuses:[],accessToken:'',siteID:1,permissions:new Set<string>(),nextPosition:0},global:{provide:{[adminPluginRegistryKey as symbol]:registry},stubs:{ElDialog:{template:'<div><slot/><slot name="footer"/></div>'}}}})
}
