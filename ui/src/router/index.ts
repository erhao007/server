import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import Login from '../views/Login.vue'
import api from '../api'

const routes: Array<RouteRecordRaw> = [
    { path: '/login', component: Login, meta: { public: true } },
    { path: '/', component: Dashboard },
    { path: '/listeners', component: () => import('../views/Listeners.vue') },
    { path: '/users', component: () => import('../views/Users.vue') },
    { path: '/storage', component: () => import('../views/Storage.vue') },
    {
        path: '/storage',
        name: 'Storage',
        component: () => import('../views/Storage.vue'),
    },
    {
        path: '/settings',
        name: 'Settings',
        component: () => import('../views/Settings.vue'),
    },
    {
        path: '/install',
        name: 'Install',
        component: () => import('../views/Install.vue'),
        meta: { public: true },
    },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

let installChecked = false;

router.beforeEach(async (to, _from, next) => {
    // Check installation status first
    if (!installChecked) {
        try {
            // We can allow the check to fail (e.g. network error) but ideally we should block
            const res = await api.checkInstall();
            if (!res.data.installed) {
                installChecked = true; // Optimization: we know it is NOT installed.
                // However, until installed, we might want to keep checking or just force to /install
                if (to.path !== '/install') {
                    return next('/install');
                }
                return next();
            }
            installChecked = true; // Installed
        } catch (e) {
            // If check fails, maybe let them proceed or show error page?
            // Proceeding might be safer to avoid loop if API is down
            console.error("Installation check failed", e);
        }
    }

    // Since we're here, assume installed (or check passed).
    // But wait, if we are cached as "installed", we should prevent visiting /install
    if (to.path === '/install') {
        const res = await api.checkInstall(); // Re-check to be sure or trust cache
        if (res.data.installed) {
            return next('/login');
        }
    }

    const token = localStorage.getItem('access_token');
    if (!to.meta.public && !token) {
        next('/login');
    } else {
        next();
    }
});

export default router;
