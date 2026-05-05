const Profile = (() => {
    let _container = null;

    const _esc = (s) => String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

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

            <div class="card-container">
                <div class="profile-grid">
                    <div class="card">
                        <div class="flex-center gap-8" style="margin-bottom: 20px;">
                            <div class="user-avatar" style="width:48px;height:48px;font-size:1.1rem;border-radius:50%;
                                 background:var(--color-accent-bg);color:var(--color-accent);
                                 display:flex;align-items:center;justify-content:center;font-weight:700;">
                                ${_esc(initials)}
                            </div>
                            <div>
                                <div style="font-weight:600;font-size:1rem;" id="displayLabel">${_esc(displayLabel)}</div>
                                <div class="text-muted" style="font-size:0.85rem;">@${_esc(user.username)}</div>
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

                        <div style="margin-top:24px;display:flex;justify-content:flex-end;">
                            <button class="btn btn-primary" id="btn-save-profile" onclick="Profile._save()">Save Changes</button>
                        </div>
                    </div>

                    <div class="card">
                        <div style="font-weight:600;font-size:1.1rem;margin-bottom:20px;">Change Password</div>
                        <div id="password-alert" class="alert"></div>

                        <div class="form-group">
                            <label class="form-label" for="profile-old-password">Current Password</label>
                            <input class="form-input" type="password" id="profile-old-password" placeholder="••••••••">
                        </div>

                        <div class="form-group">
                            <label class="form-label" for="profile-new-password">New Password</label>
                            <input class="form-input" type="password" id="profile-new-password" placeholder="At least 8 characters">
                        </div>

                        <div class="form-group">
                            <label class="form-label" for="profile-new-password-confirm">Confirm New Password</label>
                            <input class="form-input" type="password" id="profile-new-password-confirm" placeholder="Repeat new password">
                        </div>

                        <div style="margin-top:24px;display:flex;justify-content:flex-end;">
                            <button class="btn btn-primary" id="btn-change-password" onclick="Profile._changePassword()">Update Password</button>
                        </div>
                    </div>
                </div>

                <div class="card card-danger">
                    <div style="font-weight:600;font-size:1.1rem;margin-bottom:8px;color:#991b1b;">Danger Zone</div>
                    <div class="text-muted" style="font-size:0.9rem;margin-bottom:20px;">
                        Permanently delete your account and all associated expense data. This action is irreversible.
                    </div>
                    <div style="display:flex;justify-content:flex-start;">
                        <button class="btn btn-error" id="btn-close-account" onclick="Profile._openCloseAccountModal()">Close Account</button>
                    </div>
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
            // Refresh display label
            document.getElementById('displayLabel').textContent = displayName;

            // Refresh nav to show new display name
            await Nav.refreshUser();
        } else {
            const msg = await Api.handleError(res, 'Could not update profile.');
            showAlert(msg);
        }
    };

    // ── Change Password ─────────────────────────────────────────
    const _changePassword = async () => {
        const alertEl = document.getElementById('password-alert');
        const showAlert = (msg, type = 'error') => {
            alertEl.className = `alert alert-${type} is-visible`;
            alertEl.textContent = msg;
        };

        const oldPass = document.getElementById('profile-old-password');
        const newPass = document.getElementById('profile-new-password');
        const confirmPass = document.getElementById('profile-new-password-confirm');
        const btn = document.getElementById('btn-change-password');

        if (!oldPass.value || !newPass.value || !confirmPass.value) {
            showAlert('All password fields are required.');
            return;
        }

        if (newPass.value !== confirmPass.value) {
            showAlert('New passwords do not match.');
            return;
        }

        btn.disabled = true;
        btn.textContent = 'Updating…';

        const res = await Api.updatePassword(oldPass.value, newPass.value);
        btn.disabled = false;
        btn.textContent = 'Update Password';

        if (res && res.ok) {
            showAlert('Password updated successfully.', 'success');
            oldPass.value = '';
            newPass.value = '';
            confirmPass.value = '';
        } else {
            const msg = await Api.handleError(res, 'Could not update password.');
            showAlert(msg);
        }
    };

    // ── Close Account ──────────────────────────────────────────
    const _openCloseAccountModal = () => {
        const html = `
            <div class="modal-backdrop" id="close-account-modal" onclick="_modalBackdropClick(event)">
                <div class="modal" role="dialog" aria-modal="true">
                    <div class="modal-header">
                        <span class="modal-title">Close Account</span>
                        <button class="modal-close" onclick="Profile._closeModal()">&times;</button>
                    </div>
                    <div class="modal-body">
                        <p style="margin-bottom: 12px; font-weight: 600; color: var(--color-danger);">This action is irreversible.</p>
                        <p class="text-muted">Are you absolutely sure you want to permanently delete your account? All your expenses will be lost forever.</p>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" onclick="Profile._closeModal()">Cancel</button>
                        <button class="btn btn-error" id="btn-close-account-confirm" onclick="Profile._closeAccount()">Delete My Account</button>
                    </div>
                </div>
            </div>
        `;
        document.body.insertAdjacentHTML('beforeend', html);
    };

    const _closeModal = () => {
        const m = document.getElementById('close-account-modal');
        if (m) m.remove();
    };

    const _closeAccount = async () => {
        const btn = document.getElementById('btn-close-account-confirm');
        if (btn) {
            btn.disabled = true;
            btn.textContent = 'Closing Account…';
        }

        const res = await Api.closeAccount();
        if (res && res.ok) {
            window.location.href = CONFIG.APP_BASE + '/index.html';
        } else {
            if (btn) {
                btn.disabled = false;
                btn.textContent = 'Delete My Account';
            }
            const msg = await Api.handleError(res, 'Could not close account.');
            alert(msg);
        }
    };

    return { load, _save, _changePassword, _openCloseAccountModal, _closeAccount, _closeModal };
})();
