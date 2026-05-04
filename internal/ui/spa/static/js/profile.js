const Profile = (() => {
    let _container = null;

    const _esc = (s) => String(s ?? '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');

    // ── Load ───────────────────────────────────────────────────
    const load = async (container) => {
        _container = container;
        container.innerHTML = `<div style="padding:20px;color:var(--color-text-muted)"><span class="spinner"></span> Loading…</div>`;

        const res = await Api.getMe();
        if (!res || !res.ok) {
            container.innerHTML = `<div class="alert alert-error">Could not load profile.</div>`;
            return;
        }
        const user = await res.json();
        _render(user);
    };

    // ── Render ─────────────────────────────────────────────────
    const _render = (user) => {
        if (!_container) return;

        const displayLabel = (user.display_name && user.display_name.trim())
            ? user.display_name.trim() : user.username;
        const initials = displayLabel.split(/\s+/).map(w => w[0]).join('').slice(0, 2).toUpperCase();

        _container.innerHTML = `
            <div class="page-header">
                <span class="page-title">Profile</span>
            </div>

            <div class="card" style="max-width: 440px;">
                <div class="flex-center gap-8" style="margin-bottom: 16px;">
                    <div class="user-avatar" style="width:44px;height:44px;font-size:1rem;border-radius:50%;
                         background:var(--color-accent-bg);color:var(--color-accent);
                         display:flex;align-items:center;justify-content:center;font-weight:700;">
                        ${_esc(initials)}
                    </div>
                    <div>
                        <div style="font-weight:600;font-size:0.95rem;">${_esc(displayLabel)}</div>
                        <div class="text-muted" style="font-size:0.8rem;">@${_esc(user.username)}</div>
                    </div>
                </div>

                <div id="profile-alert" class="alert"></div>

                <div class="form-group">
                    <label class="form-label" for="profile-display-name">Display Name</label>
                    <input class="form-input" type="text" id="profile-display-name"
                           value="${_esc(user.display_name || '')}"
                           placeholder="Your display name (optional)"
                           maxlength="100"
                           onkeydown="if(event.key==='Enter') Profile._save()">
                </div>

                <div class="form-group mb-0">
                    <label class="form-label">Username</label>
                    <input class="form-input" type="text" value="${_esc(user.username)}" disabled
                           style="background:var(--color-bg);color:var(--color-text-muted);">
                    <div class="text-muted" style="font-size:0.75rem;margin-top:4px;">Username cannot be changed.</div>
                </div>

                <div style="margin-top:16px;display:flex;justify-content:flex-end;gap:8px;">
                    <button class="btn btn-primary" id="btn-save-profile" onclick="Profile._save()">Save Changes</button>
                </div>
            </div>
        `;

        setTimeout(() => {
            const f = document.getElementById('profile-display-name');
            if (f) f.focus();
        }, 50);
    };

    // ── Save ───────────────────────────────────────────────────
    const _save = async () => {
        const alertEl = document.getElementById('profile-alert');
        const showAlert = (msg, type = 'error') => {
            alertEl.className = `alert alert-${type} is-visible`;
            alertEl.textContent = msg;
        };

        const displayName = document.getElementById('profile-display-name').value.trim();
        const btn = document.getElementById('btn-save-profile');
        btn.disabled = true;
        btn.textContent = 'Saving…';

        const res = await Api.updateMe(displayName);
        btn.disabled = false;
        btn.textContent = 'Save Changes';

        if (res && res.ok) {
            showAlert('Profile updated.', 'success');
            // Refresh nav to show new display name
            await Nav.refreshUser();
        } else {
            const msg = await Api.handleError(res, 'Could not update profile.');
            showAlert(msg);
        }
    };

    return { load, _save };
})();
