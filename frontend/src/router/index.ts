import { createRouter, createWebHistory } from 'vue-router'
import DefaultLayout from '../layout/DefaultLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: DefaultLayout,
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'dashboard', component: () => import('../views/dashboard/index.vue') },
        { path: 'vm', name: 'vm', component: () => import('../views/vm/index.vue') },
        { path: 'vm/:name', name: 'vm-detail', component: () => import('../views/vm-detail/index.vue') },
        { path: 'snapshot', name: 'snapshot', component: () => import('../views/snapshot/index.vue') },
        { path: 'network', name: 'network', component: () => import('../views/network/index.vue') },
        { path: 'storage', name: 'storage', component: () => import('../views/storage/index.vue') },
        { path: 'iso', name: 'iso', component: () => import('../views/iso/index.vue') },
      ],
    },
    {
      path: '/vnc/:name',
      name: 'vnc',
      component: () => import('../views/vnc/index.vue'),
    },
  ],
})

export default router
