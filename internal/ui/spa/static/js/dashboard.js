const Dashboard = (() => {
    let _expenses = [];
    let _sortCol = 'date';
    let _sortDir = 'asc';
    let _container = null;

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

    // ── Load ───────────────────────────────────────────────────
    const load = async (container) => {
        _container = container;
        container.innerHTML = `<div style="padding:20px;color:var(--color-text-muted)"><span class="spinner"></span> Loading…</div>`;

        const expRes = await Api.getExpenses({ from: _firstOfMonth(), to: _today() });

        if (expRes && expRes.ok) _expenses = (await expRes.json()).expenses || [];
        else _expenses = [];

        _render();
    };

    // ── Main render ────────────────────────────────────────────
    const _render = () => {
        if (!_container) return;
        _container.innerHTML = `
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
                <button class="btn btn-secondary btn-sm" onclick="Dashboard._clearFilters()">Clear</button>
            </div>

            <div class="table-wrap">
                <table class="data-table" id="expense-table">
                    <thead>
                        <tr>
                            <th onclick="Dashboard._sort('occurred_at')" class="${_sortClass('date')}">Date</th>
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
        `;
    };

    const _sortClass = (col) =>
        _sortCol === col ? (_sortDir === 'asc' ? 'sort-asc' : 'sort-desc') : '';

    const _renderRows = () => {
        if (!_expenses.length) {
            return `<tr><td colspan="4" class="table-empty">No expenses found for this period.</td></tr>`;
        }
        const sorted = _sorted(_expenses);
        return sorted.map(e => `
            <tr>
                <td class="col-date">${_fmtDate(e.occurred_at)}</td>
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

    const _refreshRows = () => {
        const tbody = document.getElementById('expense-tbody');
        if (tbody) tbody.innerHTML = _renderRows();
    };

    // ── Sort ───────────────────────────────────────────────────
    const _sort = (col) => {
        if (_sortCol === col) {
            _sortDir = _sortDir === 'asc' ? 'desc' : 'asc';
        } else {
            _sortCol = col;
            _sortDir = col === 'amount' ? 'desc' : 'asc';
        }
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
        const from = document.getElementById('flt-from').value;
        const to = document.getElementById('flt-to').value;

        const params = {};
        if (from) params.from = from;
        if (to) params.to = to;

        const tbody = document.getElementById('expense-tbody');
        if (tbody) tbody.innerHTML = `<tr class="loading-row"><td colspan="4"><span class="spinner"></span></td></tr>`;

        const res = await Api.getExpenses(params);
        if (res && res.ok) {
            _expenses = (await res.json()).expenses || [];
        } else {
            _expenses = [];
        }
        _refreshRows();
    };

    const _clearFilters = () => {
        document.getElementById('flt-from').value = _firstOfMonth();
        document.getElementById('flt-to').value = _today();
        _applyFilters();
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

        document.body.insertAdjacentHTML('beforeend', html);
        setTimeout(() => {
            const f = document.getElementById('exp-date');
            if (f) f.focus();
        }, 50);
    };

    const _closeModal = () => {
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

        if (!date) { showErr('Please select a date.'); return; }
        if (!desc) { showErr('Please enter a description.'); return; }
        if (!amountRaw || isNaN(amountRaw) || Number(amountRaw) <= 0) {
            showErr('Please enter a valid amount.'); return;
        }

        const payload = {
            occurred_at: date + 'T00:00:00Z',
            description: desc,
            amount: Number(Number(amountRaw).toFixed(2)),
        };

        const btn = document.getElementById('btn-save-expense');
        btn.disabled = true;

        const res = (id && id != 'null')
            ? await Api.updateExpense(id, payload)
            : await Api.createExpense(payload);

        if (res && (res.ok || res.status === 201)) {
            const saved = await res.json();
            if (id) {
                const idx = _expenses.findIndex(e => e.id === id);
                if (idx >= 0) _expenses[idx] = saved; else _expenses.push(saved);
            } else {
                _expenses.push(saved);
            }
            _closeModal();
            _refreshRows();
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
            _expenses = _expenses.filter(e => e.id !== id);
            _closeModal();
            _refreshRows();
        } else {
            const msg = await Api.handleError(res, 'Could not delete expense.');
            alert(msg);
        }
    };

    return {
        load,
        _sort, _applyFilters, _clearFilters,
        _openExpenseModal, _closeModal, _saveExpense,
        _confirmDelete, _deleteExpense,
    };
})();

function _modalBackdropClick(e) {
    if (e.target === e.currentTarget) {
        const m = document.getElementById('expense-modal');
        if (m) m.remove();
        const c = document.getElementById('confirm-modal');
        if (c) c.remove();
    }
}
