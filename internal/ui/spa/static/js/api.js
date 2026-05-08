const Api = (() => {
    const _fetch = async (method, path, body) => {
        const opts = {
            method,
            credentials: 'include',
            headers: {},
        };
        if (body !== undefined) {
            opts.headers['Content-Type'] = 'application/json';
            opts.body = JSON.stringify(body);
        }
        const res = await fetch(CONFIG.API_BASE + path, opts);
        if (res.status === 401 &&
            !(path.startsWith('/api/users/signup') ||
                path.startsWith('/api/users/signin') ||
                path.startsWith('/api/users/me/password'))
        ) {
            window.location.href = CONFIG.APP_BASE + '/index.html';
            return null;
        }
        return res;
    };

    const get = (path) => _fetch('GET', path);
    const post = (path, body) => _fetch('POST', path, body);
    const put = (path, body) => _fetch('PUT', path, body);
    const patch = (path, body) => _fetch('PATCH', path, body);
    const del = (path) => _fetch('DELETE', path);

    const handleError = async (res, fallbackMsg) => {
        if (!res) return fallbackMsg;
        try {
            const text = await res.text();
            try {
                const data = JSON.parse(text);
                return data.error || data.message || 'Error: ' + text || fallbackMsg;
            } catch {
                return 'Error: ' + text || fallbackMsg;
            }
        } catch {
            return fallbackMsg;
        }
    };

    // Auth
    const signup = (username, password) =>
        post('/api/users/signup', { username, password });

    const login = (username, password) =>
        post('/api/users/signin', { username, password });

    const logout = () =>
        post('/api/users/me/signout');

    const touchSession = () =>
        post('/api/users/me/keepalive');

    // Profile
    const getMe = () =>
        get('/api/users/me');

    const updateMe = (display_name) =>
        patch('/api/users/me', { display_name });

    const updatePassword = (old_password, new_password) =>
        post('/api/users/me/password', { old_password, new_password });

    const closeAccount = () =>
        del('/api/users/me');

    // Expenses
    const getExpenses = (params = {}) => {
        const qs = new URLSearchParams();
        if (params.from) qs.set('from', params.from);
        if (params.to) qs.set('to', params.to);
        if (params.page) qs.set('page', params.page);
        if (params.page_size) qs.set('page_size', params.page_size);
        if (params.sort_by) qs.set('sort_by', params.sort_by);
        if (params.sort_desc) qs.set('sort_desc', params.sort_desc);
        if (params.category_id) qs.set('category_id', params.category_id);
        const q = qs.toString();
        return get('/api/expenses' + (q ? '?' + q : ''));
    };

    const createExpense = (expense) =>
        post('/api/expenses', expense);

    const updateExpense = (id, expense) =>
        patch(`/api/expenses/${id}`, expense);

    const deleteExpense = (id) =>
        del(`/api/expenses/${id}`);

    // Categories
    const getCategories = (prefix = '') =>
        get(`/api/categories${prefix ? '?prefix=' + encodeURIComponent(prefix) : ''}`);

    const createCategory = (name) =>
        post('/api/categories', { name });

    const updateCategory = (id, name) =>
        patch(`/api/categories/${id}`, { name });

    const deleteCategory = (id) =>
        del(`/api/categories/${id}`);

    return {
        get, post, put, del,
        handleError,
        signup, login, logout, touchSession,
        getMe, updateMe, updatePassword, closeAccount,
        getExpenses, createExpense, updateExpense, deleteExpense,
        getCategories, createCategory, updateCategory, deleteCategory,
    };
})();
