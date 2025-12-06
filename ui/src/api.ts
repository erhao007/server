import axios from 'axios';

const client = axios.create({
    baseURL: '/api/v1',
    headers: {
        'Content-Type': 'application/json',
    },
});

export interface Listener {
    id: string;
    type: string;
    address: string;
    protocol: string;
}

export interface User {
    username: string;
    client?: boolean;
    disallow: boolean;
    remarks?: string;
    is_admin?: boolean;
}

export interface LoginResponse {
    access_token: string;
    refresh_token: string;
}

// Request Interceptor
client.interceptors.request.use(config => {
    // Skip auth header for login endpoint to avoid confusion/overhead
    if (config.url?.endsWith('/login') || config.url?.endsWith('/refresh')) {
        return config;
    }

    const token = localStorage.getItem('access_token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

// Response Interceptor
client.interceptors.response.use(response => {
    return response;
}, async error => {
    const originalRequest = error.config;
    // Don't retry if it's already retried or if it's the login request itself failing
    if (error.response?.status === 401 && !originalRequest._retry && !originalRequest.url?.endsWith('/login')) {
        originalRequest._retry = true;
        const refreshToken = localStorage.getItem('refresh_token');
        if (refreshToken) {
            try {
                const res = await axios.post<LoginResponse>('/api/v1/refresh', { refresh_token: refreshToken });
                localStorage.setItem('access_token', res.data.access_token);
                localStorage.setItem('refresh_token', res.data.refresh_token);
                originalRequest.headers.Authorization = `Bearer ${res.data.access_token}`;
                return client(originalRequest);
            } catch (authError) {
                localStorage.removeItem('access_token');
                localStorage.removeItem('refresh_token');
                localStorage.removeItem('username');
                window.location.href = '/login';
                return Promise.reject(authError);
            }
        } else {
            window.location.href = '/login';
        }
    }
    return Promise.reject(error);
});

export default {
    async login(username: string, password: string) {
        const res = await client.post<LoginResponse>('/login', { username, password });
        localStorage.setItem('access_token', res.data.access_token);
        localStorage.setItem('refresh_token', res.data.refresh_token);
        localStorage.setItem('username', username);
        return res;
    },
    logout() {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        localStorage.removeItem('username');
        window.location.href = '/login';
    },
    // Install Flow
    checkInstall() {
        return client.get<{ installed: boolean }>('/install/check');
    },
    install(username: string, password: string) {
        return client.post('/install', { username, password });
    },
    getStats() {
        return client.get<any>('/stats');
    },
    getListeners() {
        return client.get<Listener[]>('/listeners');
    },
    addListener(type: string, id: string, address: string) {
        return client.post('/listeners', { type, id, address });
    },
    deleteListener(id: string) {
        return client.delete(`/listeners/${id}`);
    },
    getUsers() {
        return client.get<User[]>('/users');
    },
    addUser(username: string, password: string, allow: boolean, remarks: string, is_admin: boolean) {
        return client.post('/users', { username, password, allow, remarks, is_admin });
    },
    updateUser(username: string, password: string, allow: boolean, remarks: string, is_admin: boolean) {
        return client.put('/users', { username, password, allow, remarks, is_admin });
    },
    deleteUser(username: string) {
        return client.delete(`/users/${username}`);
    },
    // Storage APIs
    getStoredClients() {
        return client.get('/storage/clients');
    },
    deleteStoredClient(id: string) {
        return client.delete(`/storage/clients/${id}`);
    },
    getStoredSubscriptions() {
        return client.get('/storage/subscriptions');
    },
    deleteStoredSubscription(clientId: string, filter: string) {
        return client.delete(`/storage/subscriptions/`, { params: { client: clientId, filter } });
    },
    getStoredRetained() {
        return client.get('/storage/retained');
    },
    deleteStoredRetained(topic: string) {
        return client.delete(`/storage/retained/`, { params: { topic } });
    },
};
