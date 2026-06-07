import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('../layouts/MainLayout.vue'),
    children: [
      { path: '', component: () => import('../pages/CompaniesPage.vue') },
      { path: 'documents', component: () => import('../pages/DocumentsPage.vue') },
      { path: 'credentials', component: () => import('../pages/CredentialsPage.vue') },
    ],
  },
]

const router = createRouter({
  // Wails recommends Hash history or Memory history since there is no standard web server
  history: createWebHashHistory(),
  routes,
})

export default router
