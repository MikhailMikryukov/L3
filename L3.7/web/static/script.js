let currentUser = null;
let currentToken = null;
let currentUserRole = null;

// API базовый URL
const API_BASE_URL = '/warehouse';

// Инициализация
document.addEventListener('DOMContentLoaded', () => {
    // Проверяем сохраненный токен
    const token = localStorage.getItem('token');
    const user = localStorage.getItem('user');
    if (token && user) {
        currentToken = token;
        currentUser = JSON.parse(user);
        currentUserRole = currentUser.role;
        showApp();
        loadItems();
    }
});

// Функция создания юзера
async function createUser() {
    const userCreate = document.getElementById('usernameCreate');
    const username = userCreate.value;
    const roleSelect = document.getElementById('roleSelectCreate');
    const role = roleSelect.value;

    try {
        const response = await fetch('/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({username, role})
        });


        if (!response.ok) {
            throw new Error('Creation failed');
        }

        const data = await response.json();
        if (data.status === 'success') {
            alert('Юзер создан')
        }

    } catch (error) {
        console.error('Creation error:', error);
        alert('Ошибка создания');
    }
}

// Функция входа
async function login() {
    const user = document.getElementById('usernameLogin');
    const username = user.value;

    try {
        const response = await fetch('/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({username})
        });

        if (!response.ok) {
            throw new Error('Login failed');
        }

        const data = await response.json();
        currentToken = data.token;
        currentUser = data.user;
        currentUserRole = currentUser.role;

        // Сохраняем в localStorage
        localStorage.setItem('token', currentToken);
        localStorage.setItem('user', JSON.stringify(currentUser));

        showApp();
        loadItems();
    } catch (error) {
        console.error('Login error:', error);
        alert('Ошибка входа');
    }
}

// Функция выхода
function logout() {
    currentToken = null;
    currentUser = null;
    currentUserRole = null;
    localStorage.removeItem('token');
    localStorage.removeItem('user');

    document.getElementById('app').style.display = 'none';
    document.getElementById('loginPage').style.display = 'flex';
}


// Показать основное приложение
function showApp() {
    document.getElementById('loginPage').style.display = 'none';
    document.getElementById('app').style.display = 'block';
    document.getElementById('username').textContent = currentUser.username;
    document.getElementById('userRole').textContent = currentUser.role;

    // Настраиваем видимость кнопок в зависимости от роли
    const addItemBtn = document.getElementById('addItemBtn');
    if (currentUserRole === 'viewer') {
        addItemBtn.style.display = 'none';
    } else {
        addItemBtn.style.display = 'flex';
    }
}

// Переключение вкладок
function showTab(tab) {
    // Обновляем кнопки
    document.getElementById('itemsTabBtn').classList.remove('active');
    document.getElementById('historyTabBtn').classList.remove('active');
    document.getElementById(`${tab}TabBtn`).classList.add('active');

    // Обновляем контент
    document.getElementById('itemsTab').classList.remove('active');
    document.getElementById('historyTab').classList.remove('active');
    document.getElementById(`${tab}Tab`).classList.add('active');

    // Загружаем данные для вкладки
    if (tab === 'history') {
        loadHistory();
    }
}

// Загрузка товаров
async function loadItems() {
    try {
        const response = await fetch(`${API_BASE_URL}/items`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to load items');
        }

        const items = await response.json();
        renderItems(items);
    } catch (error) {
        console.error('Load items error:', error);
        document.getElementById('itemsTableBody').innerHTML = `
            <tr>
                <td colspan="6" class="loading-cell">
                    <i class="fas fa-exclamation-circle"></i> Ошибка загрузки товаров
                </td>
            </tr>
        `;
    }
}

// Отображение товаров
function renderItems(items) {
    const tbody = document.getElementById('itemsTableBody');

    if (items.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" class="loading-cell">
                    <i class="fas fa-box-open"></i> Нет товаров
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = items.map(item => `
        <tr>
            <td>${item.id}</td>
            <td><strong>${escapeHtml(item.name)}</strong></td>
            <td>${item.quantity}</td>
            <td>${item.price.toLocaleString('ru-RU')} ₽</td>
            <td>${formatDate(item.created_at)}</td>
            <td class="action-buttons">
                <button onclick="showHistory(${item.id})" class="history-btn">
                    <i class="fas fa-history"></i> История
                </button>
                ${currentUserRole !== 'viewer' ? `
                    <button onclick="editItem(${item.id})" class="edit-btn">
                        <i class="fas fa-edit"></i> Редактировать
                    </button>
                ` : ''}
                ${currentUserRole === 'admin' ? `
                    <button onclick="deleteItem(${item.id})" class="delete-btn">
                        <i class="fas fa-trash"></i> Удалить
                    </button>
                ` : ''}
            </td>
        </tr>
    `).join('');
}

// Показать модальное окно добавления товара
function showAddItemModal() {
    document.getElementById('modalTitle').textContent = 'Добавить товар';
    document.getElementById('itemId').value = '';
    document.getElementById('itemName').value = '';
    document.getElementById('itemQuantity').value = '0';
    document.getElementById('itemPrice').value = '0';
    document.getElementById('itemModal').style.display = 'block';
}

// Редактирование товара
async function editItem(id) {
    try {
        const response = await fetch(`${API_BASE_URL}/items/${id}`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to load item');
        }

        const item = await response.json();

        document.getElementById('modalTitle').textContent = 'Редактировать товар';
        document.getElementById('itemId').value = item.id;
        document.getElementById('itemName').value = item.name;
        document.getElementById('itemQuantity').value = item.quantity;
        document.getElementById('itemPrice').value = item.price;
        document.getElementById('itemModal').style.display = 'block';
    } catch (error) {
        console.error('Edit item error:', error);
        alert('Ошибка загрузки товара');
    }
}

// Сохранение товара
async function saveItem(event) {
    event.preventDefault();

    const id = document.getElementById('itemId').value;
    const item = {
        name: document.getElementById('itemName').value,
        quantity: parseInt(document.getElementById('itemQuantity').value),
        price: parseFloat(document.getElementById('itemPrice').value)
    };

    try {
        let response;
        if (id) {
            // Обновление
            response = await fetch(`${API_BASE_URL}/items/${id}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${currentToken}`
                },
                body: JSON.stringify(item)
            });
        } else {
            // Создание
            response = await fetch(`${API_BASE_URL}/items`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${currentToken}`
                },
                body: JSON.stringify(item)
            });
        }

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to save item');
        }

        closeModal();
        loadItems();
    } catch (error) {
        console.error('Save item error:', error);
        alert('Ошибка сохранения: ' + error.message);
    }
}

// Удаление товара
async function deleteItem(id) {
    if (!confirm('Вы уверены, что хотите удалить этот товар?')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/items/${id}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to delete item');
        }

        loadItems();
    } catch (error) {
        console.error('Delete item error:', error);
        alert('Ошибка удаления товара');
    }
}

// Загрузка истории
async function loadHistory() {
    const params = new URLSearchParams();
    const action = document.getElementById('filterAction').value;
    const user = document.getElementById('filterUser').value;
    const id = document.getElementById('filterItemId').value;
    const date_from = document.getElementById('filterDateFrom').value;
    const date_to = document.getElementById('filterDateTo').value;

    if (action) params.append('action', action);
    if (user) params.append('user', user);
    if (id) params.append('id', id);
    if (date_from) params.append('date_from', date_from);
    if (date_to) params.append('date_to', date_to);

    try {
        const response = await fetch(`${API_BASE_URL}/history?${params.toString()}`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`,
            }
        });

        if (!response.ok) {
            throw new Error('Failed to load history');
        }

        const history = await response.json();
        renderHistory(history);
    } catch (error) {
        console.error('Load history error:', error);
        document.getElementById('historyTableBody').innerHTML = `
            <tr>
                <td colspan="5" class="loading-cell">
                    <i class="fas fa-exclamation-circle"></i> Ошибка загрузки истории
                </td>
            </tr>
        `;
    }
}

// Очистка всех фильтров
function clearFilters() {
    document.getElementById('filterAction').value = '';
    document.getElementById('filterUser').value = '';
    document.getElementById('filterItemId').value = '';
    document.getElementById('filterDateFrom').value = '';
    document.getElementById('filterDateTo').value = '';
    loadHistory();
}

// Отображение истории
function renderHistory(history) {
    const tbody = document.getElementById('historyTableBody');

    if (history.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="5" class="loading-cell">
                    <i class="fas fa-info-circle"></i> История пуста
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = history.map(record => `
        <tr>
            <td>${formatDate(record.changed_at)}</td>
            <td>${record.item_id}</td>
            <td>
                <span class="badge badge-${record.action.toLowerCase()}">
                    ${getActionText(record.action)}
                </span>
            </td>
            <td>${escapeHtml(record.changed_by)}</td>
            <td>
                ${formatChanges(record)}
            </td>
        </tr>
    `).join('');
}

// Показать историю товара
async function showHistory(itemId) {
    try {
        const response = await fetch(`${API_BASE_URL}/items/${itemId}/history`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to load history');
        }

        const history = await response.json();

        document.getElementById('historyItemId').textContent = itemId;
        const tbody = document.getElementById('historyModalBody');

        if (history.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="4" class="loading-cell">Нет истории изменений</td>
                </tr>
            `;
        } else {
            tbody.innerHTML = history.map(record => `
                <tr>
                    <td>${formatDate(record.changed_at)}</td>
                    <td>
                        <span class="badge badge-${record.action.toLowerCase()}">
                            ${getActionText(record.action)}
                        </span>
                    </td>
                    <td>${escapeHtml(record.changed_by)}</td>
                    <td>${formatChanges(record)}</td>
                </tr>
            `).join('');
        }

        document.getElementById('historyModal').style.display = 'block';
    } catch (error) {
        console.error('Show history error:', error);
        alert('Ошибка загрузки истории');
    }
}

// Форматирование изменений
function formatChanges(record) {
    if (record.action === 'INSERT') {
        return `Создан товар: ${record.new_data?.name || ''}`;
    } else if (record.action === 'DELETE') {
        return `Удален товар: ${record.old_data?.name || ''}`;
    } else if (record.action === 'UPDATE') {
        const changes = [];
        if (record.old_data && record.new_data) {
            if (record.old_data.name !== record.new_data.name) {
                changes.push(`название: "${record.old_data.name}" → "${record.new_data.name}"`);
            }
            if (record.old_data.quantity !== record.new_data.quantity) {
                changes.push(`количество: ${record.old_data.quantity} → ${record.new_data.quantity}`);
            }
            if (record.old_data.price !== record.new_data.price) {
                changes.push(`цена: ${record.old_data.price} → ${record.new_data.price}`);
            }
        }
        return changes.length > 0 ? changes.join('; ') : 'Изменены данные';
    }
    return '-';
}

// Вспомогательные функции
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString('ru-RU');
}

function getActionText(action) {
    const actions = {
        'INSERT': 'Создание',
        'UPDATE': 'Обновление',
        'DELETE': 'Удаление'
    };
    return actions[action] || action;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function closeModal() {
    document.getElementById('itemModal').style.display = 'none';
}

function closeHistoryModal() {
    document.getElementById('historyModal').style.display = 'none';
}

// Закрытие модального окна при клике вне его
window.onclick = function (event) {
    const modal = document.getElementById('itemModal');
    const historyModal = document.getElementById('historyModal');
    if (event.target === modal) {
        closeModal();
    }
    if (event.target === historyModal) {
        closeHistoryModal();
    }
}