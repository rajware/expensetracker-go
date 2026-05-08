const Categories = (() => {
    let _categories = [];
    let _container = null;
    let _modalKeyListener = null;

    const _esc = (s) => String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

    const load = async (container) => {
        _container = container;
        container.innerHTML = `<div style="padding:20px;color:var(--color-text-muted)"><span class="spinner"></span> Loading…</div>`;

        await _fetchData();
        _render();
    };

    const _fetchData = async () => {
        const res = await Api.getCategories();
        if (res && res.ok) {
            _categories = await res.json();
        } else {
            _categories = [];
        }
    };

    const _render = () => {
        if (!_container) return;

        _container.innerHTML = `
            <div class="card-container">
                <div class="page-header">
                    <span class="page-title">Manage Categories</span>
                    <button class="btn btn-primary btn-sm" onclick="Categories._openCategoryModal(null)">+ Add Category</button>
                </div>

                <div class="card">
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th style="width: 100px">Type</th>
                                <th class="col-actions"></th>
                            </tr>
                        </thead>
                        <tbody>
                            ${_renderRows()}
                        </tbody>
                    </table>
                </div>
                
                <div class="text-muted" style="font-size: 0.8rem; padding: 0 8px;">
                    <p><strong>Note:</strong> Deleting a category will move all its expenses to the "Uncategorised" category.</p>
                </div>
            </div>
        `;
    };

    const _renderRows = () => {
        if (!_categories.length) {
            return `<tr><td colspan="3" class="table-empty">No categories found.</td></tr>`;
        }

        const systemUserID = "00000000-0000-0000-0000-000000000001";
        const uncategorisedID = "00000000-0000-0000-0000-000000000002";

        return _categories.map(c => {
            const isSystem = c.owner_id === systemUserID;
            const isUncategorised = c.id === uncategorisedID;
            const canManage = !isSystem && !isUncategorised;

            return `
                <tr>
                    <td>
                        <strong>${_esc(c.name)}</strong>
                    </td>
                    <td>
                        <span class="badge ${isSystem ? 'badge-system' : ''}">${isSystem ? 'System' : 'Custom'}</span>
                    </td>
                    <td class="col-actions">
                        ${canManage ? `
                            <button class="btn btn-ghost btn-icon btn-sm" title="Rename"
                                    onclick="Categories._openCategoryModal('${c.id}')">✎</button>
                            <button class="btn btn-ghost btn-icon btn-sm text-danger" title="Delete"
                                    onclick="Categories._confirmDelete('${c.id}')">✕</button>
                        ` : ''}
                    </td>
                </tr>
            `;
        }).join('');
    };

    const _openCategoryModal = (id) => {
        const cat = id ? _categories.find(c => c.id === id) : null;
        const isEdit = !!cat;
        const title = isEdit ? 'Rename Category' : 'Add Category';
        const nameVal = cat ? cat.name : '';

        const html = `
            <div class="modal-backdrop" id="cat-modal" onclick="_modalBackdropClick(event)">
                <div class="modal" role="dialog" aria-modal="true">
                    <div class="modal-header">
                        <span class="modal-title">${title}</span>
                        <button class="modal-close" onclick="Categories._closeModal()">&times;</button>
                    </div>
                    <div class="modal-body">
                        <div id="cat-modal-alert" class="alert"></div>
                        <div class="form-group">
                            <label class="form-label" for="cat-name">Category Name</label>
                            <input class="form-input" type="text" id="cat-name" value="${_esc(nameVal)}" 
                                   placeholder="e.g. Groceries, Travel, Entertainment" maxlength="50" required>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" onclick="Categories._closeModal()">Cancel</button>
                        <button class="btn btn-primary" id="btn-save-cat" onclick="Categories._saveCategory('${id}')">
                            ${isEdit ? 'Save Changes' : 'Add Category'}
                        </button>
                    </div>
                </div>
            </div>
        `;

        document.body.insertAdjacentHTML('beforeend', html);

        const _onKey = (e) => {
            if (e.key === 'Enter')  { e.preventDefault(); Categories._saveCategory(id); }
            if (e.key === 'Escape') { Categories._closeModal(); }
        };
        document.addEventListener('keydown', _onKey);
        _modalKeyListener = _onKey;

        setTimeout(() => document.getElementById('cat-name')?.focus(), 50);
    };

    const _closeModal = () => {
        if (_modalKeyListener) {
            document.removeEventListener('keydown', _modalKeyListener);
            _modalKeyListener = null;
        }
        const m = document.getElementById('cat-modal');
        if (m) m.remove();
        const c = document.getElementById('cat-confirm-modal');
        if (c) c.remove();
    };

    const _saveCategory = async (id) => {
        const alertEl = document.getElementById('cat-modal-alert');
        const showErr = (msg) => {
            alertEl.className = 'alert alert-error is-visible';
            alertEl.textContent = msg;
        };

        const name = document.getElementById('cat-name').value.trim();
        if (!name) { showErr('Please enter a name.'); return; }

        const btn = document.getElementById('btn-save-cat');
        btn.disabled = true;

        const res = (id && id !== 'null')
            ? await Api.updateCategory(id, name)
            : await Api.createCategory(name);

        if (res && (res.ok || res.status === 201)) {
            await _fetchData();
            _closeModal();
            _render();
        } else {
            btn.disabled = false;
            const msg = await Api.handleError(res, 'Could not save category.');
            showErr(msg);
        }
    };

    const _confirmDelete = (id) => {
        const cat = _categories.find(c => c.id === id);
        if (!cat) return;

        const html = `
            <div class="modal-backdrop" id="cat-confirm-modal" onclick="_modalBackdropClick(event)">
                <div class="modal" role="dialog" aria-modal="true">
                    <div class="modal-header">
                        <span class="modal-title">Delete Category</span>
                        <button class="modal-close" onclick="Categories._closeModal()">&times;</button>
                    </div>
                    <div class="modal-body">
                        <p class="confirm-msg">Delete category <strong>${_esc(cat.name)}</strong>?</p>
                        <p class="confirm-submsg text-muted" style="font-size: 0.8rem; margin-top: 8px;">
                            All expenses currently in this category will be moved to <strong>Uncategorised</strong>.
                        </p>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" onclick="Categories._closeModal()">Cancel</button>
                        <button class="btn btn-danger" onclick="Categories._deleteCategory('${id}')">Delete</button>
                    </div>
                </div>
            </div>
        `;
        document.body.insertAdjacentHTML('beforeend', html);
    };

    const _deleteCategory = async (id) => {
        const res = await Api.deleteCategory(id);
        if (res && (res.ok || res.status === 204)) {
            await _fetchData();
            _closeModal();
            _render();
        } else {
            const msg = await Api.handleError(res, 'Could not delete category.');
            alert(msg);
        }
    };

    return { load, _openCategoryModal, _closeModal, _saveCategory, _confirmDelete, _deleteCategory };
})();
