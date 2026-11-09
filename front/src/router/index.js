import { createRouter, createWebHashHistory } from 'vue-router'
import AppLayout from '../layouts/AppLayout.vue'
import AuthLayout from '../layouts/AuthLayout.vue'
import { pendingModules, navigation } from './navigation'
import { moduleForPath, moduleCatalog, moduleView } from '../modules/registry'
import { isModuleActive, settingsState } from '../modules/state'
import { usePageLoading } from '../composables/usePageLoading'
import { useAuth } from '../composables/useAuth'
const pageLoading = usePageLoading()
const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/', redirect: '/dashboard' },
    {
      path: '/',
      component: AuthLayout,
      children: [
        {
          path: 'login',
          name: 'login',
          component: () => import('../views/auth/LoginView.vue'),
          meta: { title: 'Sign In' },
        },
        {
          path: 'forgot-password',
          name: 'forgot-password',
          component: () => import('../views/auth/ForgotPasswordView.vue'),
          meta: { title: 'Recover Access' },
        },
        {
          path: 'reset-password',
          name: 'reset-password',
          component: () => import('../views/auth/ResetPasswordView.vue'),
          meta: { title: 'Reset Password' },
        },
      ],
    },
    {
      path: '/',
      component: AppLayout,
      meta: { requiresAuth: true },
      children: [
        ...moduleCatalog.flatMap((module) =>
          module.paths.map((path) => ({
            path: path.slice(1),
            component: moduleView(module) || (() => import('../views/system/ModuleView.vue')),
            meta: {
              title:
                navigation.flatMap((group) => group.items).find((item) => item.to === path)
                  ?.label || module.name,
              icon: module.icon,
              moduleId: module.id,
              pending: !moduleView(module),
            },
          })),
        ),
        ...pendingModules
          .filter((item) => item.to !== '/alerts' && !moduleForPath(item.to))
          .map((item) => ({
            path: item.to.slice(1),
            component: () => import('../views/system/ModuleView.vue'),
            meta: { title: item.label, icon: item.icon, pending: true },
          })),
        {
          path: 'settings',
          name: 'settings',
          component: () => import('../views/system/SettingsView.vue'),
          meta: { title: 'Settings' },
        },
        {
          path: 'module-unavailable',
          name: 'module-unavailable',
          component: () => import('../views/system/ModuleUnavailableView.vue'),
          meta: { title: 'Module Unavailable' },
        },
        {
          path: 'alerts',
          name: 'alerts',
          component: () => import('../views/system/AlertCenterView.vue'),
          meta: { title: 'Alert Center' },
        },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('../views/dashboard/DashboardView.vue'),
          meta: { title: 'Dashboard' },
        },
        {
          path: 'vehicles',
          name: 'vehicles',
          component: () => import('../views/vehicles/AssetsView.vue'),
          meta: { title: 'All Assets' },
        },
        {
          path: 'template',
          name: 'template',
          component: () => import('../views/system/TemplateView.vue'),
          meta: { title: 'Master Layout' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('../views/system/NotFoundView.vue'),
      meta: { title: 'Page Not Found' },
    },
  ],
})
for (const record of router.getRoutes()) {
  const owner = moduleForPath(record.path)
  if (owner) record.meta.moduleId = owner.id
}
router.beforeEach(async (to) => {
  pageLoading.start()
  const auth = useAuth()
  if (!auth.authDemo) {
    if (to.meta.requiresAuth && !(await auth.restore()))
      return { name: 'login', query: { redirect: to.fullPath } }
    if (to.name === 'login' && (await auth.restore())) return '/dashboard'
  }
  if (!isModuleActive(to.meta.moduleId))
    return { name: 'module-unavailable', query: { module: to.meta.moduleId } }
})
router.onError(() => pageLoading.fail())
router.afterEach((to, from, failure) => {
  pageLoading.finish()
  if (failure) return
  document.title = `${to.meta.title || 'Fleet'} · ${settingsState.value.branding.name}`
})
export default router
