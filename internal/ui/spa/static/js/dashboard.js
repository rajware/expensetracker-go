const Dashboard = (() => {
    let _expenses = [];
    let _sortCol = 'date';
    let _sortDir = 'asc';
    let _container = null;
    let _currentPage = 1;
    let _pageSize = CONFIG.PAGE_SIZE || 25;
    let _totalCount = 0;
    let _modalKeyListener = null;
    let _catSuggestions = [];
    let _selectedCatID = '';
    let _lastSelectedCatName = '';
    let _catHighlightedIdx = -1;
    let _categories = [];

    // ── Helpers ────────────────────────────────────────────────
    const _esc = (s) => String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

    const _fmtDate = (iso) => {
        if (!iso) return '';
        const d = new Date(iso);
        return d.toLocaleDateString('en-CA'); // YYYY-MM-DD
    };

    const _fmtAmount = (n) =>
        Number(n).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });

    const _today = () => new Date().toISOString().slice(0, 10);

    const _firstOfMonth = () => {
        const d = new Date();
        return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-01`;
    };

    const load = async (container) => {
        _container = container;
        container.innerHTML = `<div style="padding:20px;color:var(--color-text-muted)"><span class="spinner"></span> Loading…</div>`;

        await Promise.all([
            _fetchData(),
            _fetchCategories()
        ]);

        _render();
    };

    const _fetchCategories = async () => {
        const res = await Api.getCategories();
        if (res && res.ok) {
            _categories = await res.json();
        }
    };

    const _fetchData = async () => {
        const from = document.getElementById('flt-from')?.value || _firstOfMonth();
        const to = document.getElementById('flt-to')?.value || _today();
        const catID = document.getElementById('flt-cat')?.value || '';

        const params = {
            from, to,
            category_id: catID,
            page: _currentPage,
            page_size: _pageSize,
            sort_by: _sortCol === 'date' ? 'occurred_at' : _sortCol,
            sort_desc: _sortDir === 'desc'
        };

        const res = await Api.getExpenses(params);
        if (res && res.ok) {
            const data = await res.json();
            _expenses = data.expenses || [];
            _totalCount = data.total_count || 0;
        } else {
            _expenses = [];
            _totalCount = 0;
        }
    };

    // ── Main render ────────────────────────────────────────────
    const _render = () => {
        if (!_container) return;
        _container.innerHTML = `
            <div class="dashboard-layout">
                <div class="page-header">
                    <span class="page-title">Dashboard</span>
                    <button class="btn btn-primary btn-sm" onclick="Dashboard._openExpenseModal(null)">+ Add Expense</button>
                </div>

                <div class="toolbar" id="dash-toolbar">
                    <span class="toolbar-label">From</span>
                    <input class="form-input toolbar-date" type="date" id="flt-from"
                           value="${_firstOfMonth()}" onchange="Dashboard._applyFilters()">
                    <span class="toolbar-label">To</span>
                    <input class="form-input toolbar-date" type="date" id="flt-to"
                           value="${_today()}" onchange="Dashboard._applyFilters()">
                    <div class="toolbar-sep"></div>
                    <span class="toolbar-label">Category</span>
                    <select class="form-input" id="flt-cat" onchange="Dashboard._applyFilters()">
                        <option value="">All Categories</option>
                        ${_categories.map(c => `<option value="${c.id}">${_esc(c.name)}</option>`).join('')}
                    </select>
                    <div class="toolbar-sep"></div>
                    <button class="btn btn-secondary btn-sm" onclick="Dashboard._clearFilters()">Clear</button>
                </div>

                <div class="table-wrap">
                    <table class="data-table" id="expense-table">
                        <thead>
                            <tr>
                                <th onclick="Dashboard._sort('occurred_at')" class="${_sortClass('occurred_at')}">Date</th>
                                <th style="width: 140px">Category</th>
                                <th onclick="Dashboard._sort('description')" class="${_sortClass('description')}">Description</th>
                                <th onclick="Dashboard._sort('amount')"      class="${_sortClass('amount')} col-amount">Amount</th>
                                <th class="col-actions"></th>
                            </tr>
                        </thead>
                        <tbody id="expense-tbody">
                            ${_renderRows()}
                        </tbody>
                    </table>
                </div>

                <div id="pagination-container">
                    ${_renderPagination()}
                </div>
            </div>
        `;
    };

    const _sortClass = (col) =>
        _sortCol === col ? (_sortDir === 'asc' ? 'sort-asc' : 'sort-desc') : '';

    const _renderRows = () => {
        if (!_expenses.length) {
            return `<tr><td colspan="4" class="table-empty">No expenses found for this period.</td></tr>`;
        }
        return _expenses.map(e => `
            <tr id="exp-row-${e.id}">
                <td class="col-date">${_fmtDate(e.occurred_at)}</td>
                <td><span class="badge">${_esc(e.category_name || 'Uncategorised')}</span></td>
                <td>${_esc(e.description || '')}</td>
                <td class="col-amount">${_fmtAmount(e.amount)}</td>
                <td class="col-actions">
                    <button class="btn btn-ghost btn-icon btn-sm" title="Edit"
                            onclick="Dashboard._openExpenseModal('${e.id}')">✎</button>
                    <button class="btn btn-ghost btn-icon btn-sm text-danger" title="Delete"
                            onclick="Dashboard._confirmDelete('${e.id}')">✕</button>
                </td>
            </tr>
        `).join('');
    };

    const _renderPagination = () => {
        const start = (_currentPage - 1) * _pageSize + 1;
        const end = Math.min(_currentPage * _pageSize, _totalCount);
        const hasPrev = _currentPage > 1;
        const hasNext = end < _totalCount;

        return `
            <div class="pagination">
                <div class="pagination-info">
                    Showing <strong>${start}-${end}</strong> of <strong>${_totalCount}</strong> expenses
                </div>
                <div class="pagination-controls">
                    <button class="btn btn-secondary btn-sm" ${!hasPrev ? 'disabled' : ''} onclick="Dashboard._prevPage()">Previous</button>
                    <button class="btn btn-secondary btn-sm" ${!hasNext ? 'disabled' : ''} onclick="Dashboard._nextPage()">Next</button>
                </div>
            </div>
        `;
    };

    const _prevPage = async () => {
        if (_currentPage > 1) {
            _currentPage--;
            await _fetchData();
            _refreshRows();
        }
    };

    const _nextPage = async () => {
        const maxPage = Math.ceil(_totalCount / _pageSize);
        if (_currentPage < maxPage) {
            _currentPage++;
            await _fetchData();
            _refreshRows();
        }
    };

    const _refreshRows = () => {
        const tbody = document.getElementById('expense-tbody');
        if (tbody) tbody.innerHTML = _renderRows();
        
        const pagContainer = document.getElementById('pagination-container');
        if (pagContainer) pagContainer.innerHTML = _renderPagination();
    };

    // ── Sort ───────────────────────────────────────────────────
    const _sort = async (col) => {
        if (_sortCol === col) {
            _sortDir = _sortDir === 'asc' ? 'desc' : 'asc';
        } else {
            _sortCol = col;
            _sortDir = col === 'amount' ? 'desc' : 'asc';
        }
        _currentPage = 1;
        await _fetchData();
        _render();
    };

    const _sorted = (arr) => {
        return [...arr].sort((a, b) => {
            let va = a[_sortCol], vb = b[_sortCol];
            if (_sortCol === 'amount') { va = Number(va); vb = Number(vb); }
            if (va < vb) return _sortDir === 'asc' ? -1 : 1;
            if (va > vb) return _sortDir === 'asc' ? 1 : -1;
            return 0;
        });
    };

    // ── Filters ────────────────────────────────────────────────
    const _applyFilters = async () => {
        _currentPage = 1; // Reset to page 1 on new filter
        const tbody = document.getElementById('expense-tbody');
        if (tbody) tbody.innerHTML = `<tr class="loading-row"><td colspan="4"><span class="spinner"></span></td></tr>`;

        await _fetchData();
        _refreshRows();
    };

    const _clearFilters = () => {
        document.getElementById('flt-from').value = _firstOfMonth();
        document.getElementById('flt-to').value = _today();
        document.getElementById('flt-cat').value = '';
        _applyFilters();
    };

    const _refreshFilterDropdown = () => {
        const select = document.getElementById('flt-cat');
        if (!select) return;
        const currentVal = select.value;
        let html = '<option value="">All Categories</option>';
        _categories.forEach(c => {
            html += `<option value="${c.id}">${_esc(c.name)}</option>`;
        });
        select.innerHTML = html;
        select.value = currentVal;
    };

    // ── Expense Modal ─────────────────────────────────────────
    const _openExpenseModal = (id) => {
        const expense = id ? _expenses.find(e => e.id === id) : null;
        const isEdit = !!expense;
        const title = isEdit ? 'Edit Expense' : 'Add Expense';

        const dateVal = expense ? _fmtDate(expense.occurred_at) : _today();
        const desc = expense ? (expense.description || '') : '';
        const amount = expense ? expense.amount : '';

        const html = `
            <div class="modal-backdrop" id="expense-modal" onclick="_modalBackdropClick(event)">
                <div class="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title">
                    <div class="modal-header">
                        <span class="modal-title" id="modal-title">${title}</span>
                        <button class="modal-close" onclick="Dashboard._closeModal()" aria-label="Close">&times;</button>
                    </div>
                    <div class="modal-body">
                        <div id="modal-alert" class="alert"></div>
                        <div class="form-group">
                            <label class="form-label" for="exp-date">Date</label>
                            <input class="form-input" type="date" id="exp-date" value="${dateVal}" required>
                        </div>
                        <div class="form-group">
                            <label class="form-label" for="exp-desc">Description</label>
                            <input class="form-input" type="text" id="exp-desc"
                                   value="${_esc(desc)}" maxlength="200">
                        </div>
                        <div class="form-group">
                            <label class="form-label" for="exp-amount">Amount</label>
                            <input class="form-input" type="number" id="exp-amount"
                                   value="${amount}" min="0.01" step="0.01"
                                   placeholder="0.00" required>
                        </div>
                        <div class="form-group">
                            <label class="form-label" for="exp-cat">Category</label>
                            <div class="cat-wrap is-selected">
                                <input class="form-input" type="text" id="exp-cat"
                                       value="${_esc(expense ? expense.category_name : 'Uncategorised')}"
                                       autocomplete="off" placeholder="Type to search or create...">
                                <div class="cat-dropdown" id="cat-suggestions" hidden></div>
                            </div>
                            <input type="hidden" id="exp-cat-id" value="${expense ? expense.category_id : '00000000-0000-0000-0000-000000000002'}">
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" onclick="Dashboard._closeModal()">Cancel</button>
                        <button class="btn btn-primary" id="btn-save-expense"
                                onclick="Dashboard._saveExpense('${id}')">
                            ${isEdit ? 'Save Changes' : 'Add Expense'}
                        </button>
                    </div>
                </div>
            </div>
        `;
        _lastSelectedCatName = expense ? expense.category_name : 'Uncategorised';

        document.body.insertAdjacentHTML('beforeend', html);

        const _onKey = (e) => {
            if (e.key === 'Enter')  { e.preventDefault(); Dashboard._saveExpense(id); }
            if (e.key === 'Escape') { Dashboard._closeModal(); }
        };
        document.addEventListener('keydown', _onKey);
        _modalKeyListener = _onKey;

        setTimeout(() => {
            const f = document.getElementById('exp-date');
            if (f) f.focus();
        }, 50);

        // Wire up autocomplete
        const catInput = document.getElementById('exp-cat');
        if (catInput) {
            catInput.addEventListener('input', (e) => Dashboard._onCatInput(e.target.value));
            catInput.addEventListener('keydown', (e) => Dashboard._onCatKey(e));
            catInput.addEventListener('focus', (e) => {
                Dashboard._onCatInput(e.target.value, true);
            });
        }
    };

    const _onCatInput = async (val, force = false) => {
        _catHighlightedIdx = -1;
        document.querySelector('.cat-wrap')?.classList.remove('is-selected');

        const dropdown = document.getElementById('cat-suggestions');
        const query = val.trim();
        
        // Show all if empty or "Uncategorised" when forced (on focus)
        const isDefault = !query || query.toLowerCase() === 'uncategorised';
        if (isDefault && !force) {
            dropdown.hidden = true;
            return;
        }

        const res = await Api.getCategories(isDefault ? '' : query);
        if (res && res.ok) {
            _catSuggestions = await res.json();
            _renderCatSuggestions(val, dropdown);
        }
    };

    const _renderCatSuggestions = (typed, dropdown) => {
        const query = typed.trim();
        const exactMatch = _catSuggestions.find(c => c.name.toLowerCase() === query.toLowerCase());
        
        let html = '';
        _catSuggestions.forEach((c, i) => {
            html += `<div class="cat-option" onclick="Dashboard._selectCat('${c.id}', '${_esc(c.name)}')">${_esc(c.name)}</div>`;
        });

        if (query && query.toLowerCase() !== 'uncategorised' && !exactMatch) {
            html += `<div class="cat-option cat-option--create" onclick="Dashboard._createCat('${_esc(query)}')">
                        + Create new category "<strong>${_esc(query)}</strong>"
                    </div>`;
        }

        if (!html) {
            html = `<div class="cat-option cat-option--empty">No matches found.</div>`;
        }

        dropdown.innerHTML = html;
        dropdown.hidden = false;
    };

    const _onCatKey = (e) => {
        const dropdown = document.getElementById('cat-suggestions');
        if (dropdown.hidden) return;

        const options = dropdown.querySelectorAll('.cat-option');
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            _catHighlightedIdx = (_catHighlightedIdx + 1) % options.length;
            _highlightOption(options);
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            _catHighlightedIdx = (_catHighlightedIdx - 1 + options.length) % options.length;
            _highlightOption(options);
        } else if (e.key === 'Enter' && _catHighlightedIdx >= 0) {
            e.preventDefault();
            options[_catHighlightedIdx].click();
        }
    };

    const _highlightOption = (options) => {
        options.forEach((opt, i) => {
            opt.classList.toggle('highlighted', i === _catHighlightedIdx);
            if (i === _catHighlightedIdx) opt.scrollIntoView({ block: 'nearest' });
        });
    };

    const _selectCat = (id, name) => {
        document.getElementById('exp-cat').value = name;
        document.getElementById('exp-cat-id').value = id;
        _lastSelectedCatName = name;
        document.getElementById('cat-suggestions').hidden = true;
        document.querySelector('.cat-wrap')?.classList.add('is-selected');
    };

    const _createCat = async (name) => {
        const res = await Api.createCategory(name);
        if (res && (res.ok || res.status === 201)) {
            const cat = await res.json();
            _categories.push(cat);
            _refreshFilterDropdown();
            _selectCat(cat.id, cat.name);
        } else {
            const msg = await Api.handleError(res, 'Could not create category.');
            alert(msg);
        }
    };

    const _closeModal = () => {
        if (_modalKeyListener) {
            document.removeEventListener('keydown', _modalKeyListener);
            _modalKeyListener = null;
        }
        const m = document.getElementById('expense-modal');
        if (m) m.remove();
        const c = document.getElementById('confirm-modal');
        if (c) c.remove();
    };

    // ── Save expense ───────────────────────────────────────────
    const _saveExpense = async (id) => {
        const alertEl = document.getElementById('modal-alert');
        const showErr = (msg) => {
            alertEl.className = 'alert alert-error is-visible';
            alertEl.textContent = msg;
        };

        const date = document.getElementById('exp-date').value;
        const desc = document.getElementById('exp-desc').value.trim();
        const amountRaw = document.getElementById('exp-amount').value;
        let catID = document.getElementById('exp-cat-id').value;
        const catName = document.getElementById('exp-cat').value.trim();

        if (!date) { showErr('Please select a date.'); return; }
        if (!desc) { showErr('Please enter a description.'); return; }
        if (!amountRaw || isNaN(amountRaw) || Number(amountRaw) <= 0) {
            showErr('Please enter a valid amount.'); return;
        }

        // If the name in the box doesn't match the ID we have, and it's not Uncategorised,
        // we should warn the user that they need to select or create a category.
        // (Unless they just cleared it, then we default to Uncategorised).
        if (!catName) {
            catID = "00000000-0000-0000-0000-000000000002";
        } else if (catName !== _lastSelectedCatName) {
            // Check if it's an exact match for an existing category
            const match = _categories.find(c => c.name.toLowerCase() === catName.toLowerCase());
            if (match) {
                catID = match.id;
                _lastSelectedCatName = match.name; // optional but good for consistency
                document.getElementById('exp-cat').value = match.name; // normalize casing
                document.querySelector('.cat-wrap')?.classList.add('is-selected');
            } else {
                showErr('Category not found. Please select from the list or create it.');
                return;
            }
        }

        const payload = {
            occurred_at: date + 'T00:00:00Z',
            description: desc,
            amount: Number(Number(amountRaw).toFixed(2)),
            category_id: catID
        };

        const btn = document.getElementById('btn-save-expense');
        btn.disabled = true;

        const res = (id && id != 'null')
            ? await Api.updateExpense(id, payload)
            : await Api.createExpense(payload);

        if (res && (res.ok || res.status === 201)) {
            const saved = await res.json();
            if (!id || id === 'null') {
                // New expense: find where it landed and go there
                await _navigateToExpense(saved.id);
            } else {
                await _fetchData();
            }
            _closeModal();
            _refreshRows();
            
            // Subtle highlight
            const row = document.getElementById(`exp-row-${saved.id}`);
            if (row) {
                row.style.background = 'var(--color-accent-bg)';
                setTimeout(() => row.style.background = '', 2000);
            }
        } else {
            btn.disabled = false;
            const msg = await Api.handleError(res, 'Could not save expense.');
            showErr(msg);
        }
    };

    // ── Delete ─────────────────────────────────────────────────
    const _confirmDelete = (id) => {
        const expense = _expenses.find(e => e.id === id);
        if (!expense) return;
        const html = `
            <div class="modal-backdrop" id="confirm-modal" onclick="_modalBackdropClick(event)">
                <div class="modal" role="dialog" aria-modal="true">
                    <div class="modal-header">
                        <span class="modal-title">Delete Expense</span>
                        <button class="modal-close" onclick="Dashboard._closeModal()">&times;</button>
                    </div>
                    <div class="modal-body">
                        <p class="confirm-msg">Delete <strong>${_esc(expense.description)}</strong>
                        for <strong>${_fmtAmount(expense.amount)}</strong> on ${_fmtDate(expense.occurred_at)}?
                        This cannot be undone.</p>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" onclick="Dashboard._closeModal()">Cancel</button>
                        <button class="btn btn-danger" onclick="Dashboard._deleteExpense('${id}')">Delete</button>
                    </div>
                </div>
            </div>
        `;
        document.body.insertAdjacentHTML('beforeend', html);
    };

    const _deleteExpense = async (id) => {
        const res = await Api.deleteExpense(id);
        if (res && (res.ok || res.status === 204)) {
            await _fetchData();
            // If we deleted the last item on the current page, go back
            if (_expenses.length === 0 && _currentPage > 1) {
                _currentPage--;
                await _fetchData();
            }
            _closeModal();
            _refreshRows();
        } else {
            const msg = await Api.handleError(res, 'Could not delete expense.');
            alert(msg);
        }
    };

    const _navigateToExpense = async (targetId) => {
        const from = document.getElementById('flt-from')?.value || _firstOfMonth();
        const to = document.getElementById('flt-to')?.value || _today();

        // Fetch everything (no page size) to find rank
        const res = await Api.getExpenses({
            from, to,
            sort_by: _sortCol === 'date' ? 'occurred_at' : _sortCol,
            sort_desc: _sortDir === 'desc',
            page_size: 0 // return all
        });

        if (res && res.ok) {
            const data = await res.json();
            const idx = data.expenses.findIndex(e => e.id === targetId);
            if (idx >= 0) {
                _currentPage = Math.floor(idx / _pageSize) + 1;
                await _fetchData();
            }
        }
    };

    return {
        load,
        _sort, _applyFilters, _clearFilters,
        _openExpenseModal, _closeModal, _saveExpense,
        _confirmDelete, _deleteExpense,
        _prevPage, _nextPage,
        _onCatInput, _onCatKey, _selectCat, _createCat
    };
})();

function _modalBackdropClick(e) {
    if (e.target === e.currentTarget) {
        e.target.remove();
    }
}
