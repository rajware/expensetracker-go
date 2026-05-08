const Nav = (() => {
    let _user = null;
    let _heartbeatTimer = null;
    let _currentPage = null;

    const pages = {
        dashboard: { label: 'Dashboard', file: 'dashboard' },
        categories: { label: 'Categories', file: 'categories' },
        profile: { label: 'Profile', file: 'profile' },
    };

    const init = async (activePage) => {
        _currentPage = activePage;
        const res = await Api.getMe();
        if (!res || !res.ok) {
            window.location.href = CONFIG.APP_BASE + '/login.html';
            return;
        }
        _user = await res.json();
        _render();
        _startHeartbeat();
    };

    const getUser = () => _user;

    const _render = () => {
        const nav = document.getElementById('main-nav');
        if (!nav) return;

        nav.style.setProperty('--nav-bg', CONFIG.NAV_BG);
        document.documentElement.style.setProperty('--color-brand', CONFIG.NAV_BG);

        const displayLabel = (_user.display_name && _user.display_name.trim())
            ? _user.display_name.trim()
            : _user.username;

        const initials = displayLabel.split(/\s+/).map(w => w[0]).join('').slice(0, 2).toUpperCase();

        nav.innerHTML = `
            <div class="nav-left">
                <a class="nav-brand" href="#" onclick="navigate('dashboard'); return false;">Expense Tracker</a>
            </div>
            <div class="nav-right">
                <a class="nav-link ${_currentPage === 'dashboard' ? 'active' : ''}"
                   href="#" onclick="navigate('dashboard'); return false;">Dashboard</a>
                <a class="nav-link ${_currentPage === 'categories' ? 'active' : ''}"
                   href="#" onclick="navigate('categories'); return false;">Categories</a>
                <div class="user-chip-wrap" id="user-chip-wrap">
                    <button class="user-chip" id="user-chip" onclick="Nav._toggleDropdown()">
                        <span class="user-avatar">${initials}</span>
                        <span class="user-label">${_esc(displayLabel)}</span>
                        <span class="chevron">&#8964;</span>
                    </button>
                    <div class="user-dropdown" id="user-dropdown" hidden>
                        <a class="dropdown-item" href="#" onclick="Nav._closeDropdown(); navigate('profile'); return false;">Edit Profile</a>
                        <div class="dropdown-divider"></div>
                        <a class="dropdown-item dropdown-item--danger" href="#" onclick="Nav._logout(); return false;">Log Out</a>
                    </div>
                </div>
            </div>
        `;

        document.addEventListener('click', _outsideClick);
    };

    const _toggleDropdown = () => {
        const dd = document.getElementById('user-dropdown');
        if (!dd) return;
        dd.hidden = !dd.hidden;
    };

    const _closeDropdown = () => {
        const dd = document.getElementById('user-dropdown');
        if (dd) dd.hidden = true;
    };

    const _outsideClick = (e) => {
        const wrap = document.getElementById('user-chip-wrap');
        if (wrap && !wrap.contains(e.target)) _closeDropdown();
    };

    const _logout = async () => {
        _closeDropdown();
        await Api.logout();
        window.location.href = CONFIG.APP_BASE + '/login.html';
    };

    const _startHeartbeat = () => {
        if (_heartbeatTimer) clearInterval(_heartbeatTimer);
        _heartbeatTimer = setInterval(async () => {
            const res = await Api.touchSession();
            if (!res) clearInterval(_heartbeatTimer);
        }, CONFIG.SESSION_TOUCH_INTERVAL_MS);
    };

    const setActivePage = (page) => {
        _currentPage = page;
        document.querySelectorAll('.nav-link').forEach(el => {
            el.classList.toggle('active', el.textContent.trim().toLowerCase() === page);
        });
    };

    const refreshUser = async () => {
        const res = await Api.getMe();
        if (res && res.ok) {
            _user = await res.json();
            _render();
        }
    };

    const _esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

    return { init, getUser, refreshUser, setActivePage, _toggleDropdown, _closeDropdown, _logout };
})();

function navigate(page) {
    Nav.setActivePage(page);
    const content = document.getElementById('page-content');
    if (!content) return;

    if (typeof PageLoaders !== 'undefined' && PageLoaders[page]) {
        content.innerHTML = '';
        PageLoaders[page](content);
    }
}
