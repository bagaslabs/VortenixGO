document.addEventListener('DOMContentLoaded', () => {
    // --- WebSocket Implementation ---
    let socket = null;
    let selectedBotId = null;
    let bots = [];
    let botDebugLogs = {}; // logStore: { botId: [array of html entries] }
    let saveTimeout = null;

    // --- Item Cache System ---
    const itemCache = {
        byId: new Map(),
        byName: new Map()
    };

    window.getItem = (query) => {
        if (!query && query !== 0) return null;

        let item = null;
        if (typeof query === 'number') {
            item = itemCache.byId.get(query);
        } else if (typeof query === 'string') {
            const id = itemCache.byName.get(query.toLowerCase());
            if (id !== undefined) item = itemCache.byId.get(id);
        }

        if (!item) {
            console.log(`[Cache Miss] requesting item: ${query}`);
            const payload = typeof query === 'number' ? { id: query } : { name: query };
            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.send(JSON.stringify({ type: 'GET_ITEM', data: payload }));
            }
            return null; // Async fetch triggered
        }
        return item; // Synchronous return if cached
    };

    function cacheItem(item) {
        if (!item) return;
        itemCache.byId.set(item.ID, item);
        if (item.Name) itemCache.byName.set(item.Name.toLowerCase(), item.ID);
    }

    function connectWS() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

        socket.onopen = () => {
            console.log('Connected to VortenixGO Server');
        };

        socket.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            if (msg.type === 'UPDATE_LIST') {
                const newBots = msg.data || [];
                const activeIds = newBots.map(b => b.id);

                const selectedWasRemoved = selectedBotId && !activeIds.includes(selectedBotId);

                Object.keys(botDebugLogs).forEach(id => {
                    if (!activeIds.includes(id)) delete botDebugLogs[id];
                });

                bots = newBots;
                renderBotList();

                if (selectedWasRemoved) {
                    deselectBot();
                } else if (selectedBotId) {
                    const updatedBot = bots.find(b => b.id === selectedBotId);
                    if (updatedBot) updateBotDashboard(updatedBot);
                }
            } else if (msg.type === 'ERROR') {
                alert('Error: ' + msg.data);
            } else if (msg.type === 'DEBUG_LOG') {
                appendDebugLog(msg.data);
            } else if (msg.type === 'ITEMS_DATA') {
                // Bulk cache search results
                if (msg.data && Array.isArray(msg.data)) {
                    msg.data.forEach(item => cacheItem(item));
                }
                renderDBItemList(msg.data);
            } else if (msg.type === 'ITEM_DATA') {
                cacheItem(msg.data);
                renderItemDetail(msg.data);
            } else if (msg.type === 'DATABASE_INFO') {
                renderDatabaseInfo(msg.data);
            }
        };

        socket.onclose = () => {
            console.log('Disconnected from Server. Retrying...');
            setTimeout(connectWS, 2000);
        };
    }

    connectWS();

    // --- UI Selection & Updates ---
    const botListContainer = document.getElementById('bot-list');
    const botCountSpan = document.getElementById('bot-count');
    const botDashboard = document.getElementById('bot-dashboard');
    const botNoSelection = document.querySelector('#bot-detail-panel .no-selection');
    const debugLogsContainer = document.getElementById('debug-logs');

    function renderBotList() {
        botListContainer.innerHTML = '';
        botCountSpan.textContent = bots ? bots.length : 0;

        if (!bots || bots.length === 0) {
            botListContainer.innerHTML = '<div class="empty-state">No bots added</div>';
            return;
        }

        bots.forEach(bot => {
            const el = document.createElement('div');
            el.className = `bot-item ${selectedBotId === bot.id ? 'active' : ''}`;
            const nameToShow = bot.display_name || bot.name;
            const subTitle = (bot.ingame_name && bot.email) ? bot.email : bot.type;

            let badgeClass = 'idle';
            const statusTextStr = bot.status || 'Idle';
            const s = statusTextStr.toLowerCase();

            if (s === 'online') badgeClass = 'online';
            else if (s.includes('connect') || s.includes('get')) badgeClass = 'warning';
            else if (s === 'http_block' || s === 'offline') badgeClass = 'danger';

            el.innerHTML = `
                <div class="info">
                    <strong>${nameToShow}</strong>
                    <div style="font-size: 0.8rem; opacity: 0.7;">${subTitle}</div>
                </div>
                <div class="status-indicator">
                    <span class="status-badge ${badgeClass}">${statusTextStr}</span>
                </div>
            `;
            el.onclick = () => selectBot(bot.id);
            botListContainer.appendChild(el);
        });
    }

    function selectBot(id) {
        selectedBotId = id;
        const bot = bots.find(b => b.id === id);
        if (bot) {
            botNoSelection.classList.add('hidden');
            botDashboard.classList.remove('hidden');
            updateBotDashboard(bot);
            renderBotList();
        }
    }

    function deselectBot() {
        selectedBotId = null;
        botNoSelection.classList.remove('hidden');
        botDashboard.classList.add('hidden');
        document.getElementById('bot-console').innerHTML = '<div class="log-line system">> Console cleared...</div>';
        debugLogsContainer.innerHTML = '';
        renderBotList();
    }

    function updateBotDashboard(bot) {
        const nameToShow = bot.display_name || bot.name;
        document.getElementById('detail-name').textContent = nameToShow;

        refreshDebugView(bot.id);

        let emailLabel = document.getElementById('detail-email-label');
        if (bot.email && bot.ingame_name) {
            if (!emailLabel) {
                emailLabel = document.createElement('div');
                emailLabel.id = 'detail-email-label';
                emailLabel.style.fontSize = '0.8rem';
                emailLabel.style.color = 'var(--text-muted)';
                emailLabel.style.marginTop = '-5px';
                document.getElementById('detail-name').parentNode.insertBefore(emailLabel, document.getElementById('detail-name').nextSibling);
            }
            emailLabel.textContent = bot.email;
            emailLabel.style.display = 'block';
        } else if (emailLabel) {
            emailLabel.style.display = 'none';
        }

        document.getElementById('detail-level-gems').textContent = `Lvl ${bot.level || 0} | ${bot.gems || 0} Gems`;
        document.getElementById('detail-world-pos').textContent = `${bot.world || 'EMPTY'} (${bot.pos_x || 0}, ${bot.pos_y || 0})`;

        document.getElementById('detail-played-age').textContent = `${bot.play_time || '0h'} | ${bot.age || 0}d`;
        document.getElementById('detail-ping').textContent = `${bot.ping || 0} ms`;
        document.getElementById('debug-enet').checked = bot.show_enet || false;

        const glogGroup = document.getElementById('detail-glog-group');
        const glogInput = document.getElementById('detail-glog');
        const proxyInput = document.getElementById('detail-proxy');

        if (bot.type === 'gmail' || bot.type === 'apple') {
            glogGroup.classList.remove('hidden');
            if (document.activeElement !== glogInput) glogInput.value = bot.glog || '';
        } else {
            glogGroup.classList.add('hidden');
        }

        if (document.activeElement !== proxyInput) proxyInput.value = bot.proxy || '';

        const statusText = document.getElementById('detail-status-text');
        const statusDot = document.getElementById('status-dot');
        const btnConnect = document.getElementById('btn-connect');
        const btnDisconnect = document.getElementById('btn-disconnect');

        if (statusText && statusDot) {
            const currentStatus = bot.status || 'Idle';
            statusText.textContent = currentStatus;
            const s = currentStatus.toLowerCase();

            // Connect is almost always enabled to allow "Force Connect/Reconnect"
            btnConnect.disabled = false;
            btnDisconnect.disabled = !bot.connected;

            if (s.includes('connect') || s.includes('get')) {
                statusText.style.color = 'var(--warning)';
                statusDot.className = 'status-dot warning';
                btnDisconnect.disabled = false;
            } else if (s === 'online') {
                statusText.style.color = 'var(--success)';
                statusDot.className = 'status-dot online';
                btnDisconnect.disabled = false;
            } else if (s === 'http_block' || s === 'offline' || s === 'disconnected' || s === 'idle') {
                statusText.style.color = 'var(--danger)';
                statusDot.className = 'status-dot offline';
                btnDisconnect.disabled = true;
                if (s === 'idle') statusText.style.color = 'var(--text-muted)';
            } else {
                statusText.style.color = 'var(--text-muted)';
                statusDot.className = 'status-dot offline';
            }
        }

        const devJson = document.getElementById('dev-internal-json');
        if (devJson) {
            // Create a copy without inventory for display
            const { local, ...rest } = bot;
            const cleanLocal = { ...local };
            delete cleanLocal.inventory;
            const displayBot = { ...rest, local: cleanLocal };
            devJson.textContent = JSON.stringify(displayBot, null, 4);
        }

        renderInventory(bot);
        renderPlayers(bot);

        // Update world map if on map tab
        const mapTab = document.getElementById('tab-map');
        if (mapTab && mapTab.classList.contains('active')) {
            if (typeof renderWorldMap === 'function') renderWorldMap(bot);
            if (typeof updateGrowscan === 'function') updateGrowscan(bot);
        }
    }

    function renderPlayers(bot) {
        const countEl = document.getElementById('player-count');
        const bodyEl = document.getElementById('player-list-body');

        if (!countEl || !bodyEl) return;

        const players = bot.local.players || [];
        countEl.textContent = players.length;

        bodyEl.innerHTML = '';
        if (players.length === 0) {
            bodyEl.innerHTML = '<tr><td colspan="2" class="text-center" style="padding: 40px; opacity: 0.5;">No players in range</td></tr>';
            return;
        }

        players.forEach(p => {
            const row = document.createElement('tr');
            row.className = 'inv-row';

            const leftCell = document.createElement('td');
            leftCell.className = 'inv-info-cell';
            leftCell.innerHTML = `
                <div style="display: flex; flex-direction: column;">
                    <span class="inv-name" style="color: ${p.is_local ? 'var(--primary)' : 'white'}; font-weight: 500;">
                        ${p.name}
                        ${p.is_local ? '<span style="color: var(--primary); opacity: 0.8; font-size: 0.8em; margin-left: 5px;">(YOU)</span>' : ''}
                        ${p.mod ? '<span style="color: var(--danger); opacity: 0.8; font-size: 0.8em; margin-left: 5px;">[MOD]</span>' : ''}
                    </span>
                    <span style="font-size: 0.75em; opacity: 0.5; margin-top: 2px;">NetID: ${p.netid} • UserID: ${p.userid}</span>
                </div>
            `;

            const rightCell = document.createElement('td');
            rightCell.className = 'inv-action-cell';
            rightCell.style.textAlign = 'right';
            rightCell.innerHTML = `
                <div style="display: flex; flex-direction: column; align-items: flex-end;">
                    <span style="font-size: 0.9em;">${p.country || 'Unknown'}</span>
                    <span style="font-size: 0.75em; opacity: 0.5; margin-top: 2px; font-family: monospace;">
                        X: ${Math.round(p.pos_x || 0)} Y: ${Math.round(p.pos_y || 0)}
                    </span>
                </div>
            `;

            row.appendChild(leftCell);
            row.appendChild(rightCell);
            bodyEl.appendChild(row);
        });
    }

    function renderInventory(bot) {
        const slotsEl = document.getElementById('inv-slots');
        const amountEl = document.getElementById('inv-amount');
        const bodyEl = document.getElementById('inventory-body');

        if (!slotsEl || !bodyEl) return;

        const inv = bot.local.inventory || [];
        const maxSlots = bot.local.inventory_slots || 0;

        slotsEl.innerHTML = `<span class="stat-value">${inv.length}</span><span style="opacity: 0.5; margin: 0 4px;">/</span><span class="stat-value" style="opacity: 0.7;">${maxSlots}</span>`;
        amountEl.innerHTML = `<span class="stat-value" style="color: var(--warning);">${(bot.local.gem_count || 0).toLocaleString()}</span>`;

        bodyEl.innerHTML = '';
        if (inv.length === 0) {
            bodyEl.innerHTML = '<tr><td colspan="2" class="text-center" style="padding: 40px; opacity: 0.5;">Inventory is empty</td></tr>';
            return;
        }

        inv.forEach((item) => {
            const row = document.createElement('tr');
            row.className = 'inv-row';

            // Try to get name and info from cache
            const dbItem = window.getItem(item.id);
            const name = item.name || (dbItem ? dbItem.Name : `Item`);
            const clothingType = dbItem ? dbItem.ClothingType : 0;
            const isWearable = clothingType !== 0;

            // Button Logic
            let buttonsHtml = '';

            // Wear/Unwear (Only if wearable)
            if (isWearable) {
                if (item.is_active) {
                    buttonsHtml += `<button class="inv-list-btn unwear" onclick="window.handleInventoryAction('${bot.id}', ${item.id}, 'UNWEAR')">Unwear</button>`;
                } else {
                    buttonsHtml += `<button class="inv-list-btn wear" onclick="window.handleInventoryAction('${bot.id}', ${item.id}, 'WEAR')">Wear</button>`;
                }
            }

            // Drop & Trash (Skip for Fist/Wrench for safety/aesthetics, though user asked for "every item". 
            // Image shows Trash for Fist but not Drop. I will just render them for standard items.)
            // IDs: 18=Fist, 32=Wrench
            if (item.id !== 18 && item.id !== 32) {
                buttonsHtml += `<button class="inv-list-btn drop" onclick="window.handleInventoryAction('${bot.id}', ${item.id}, 'DROP')">Drop</button>`;
            }
            buttonsHtml += `<button class="inv-list-btn trash" onclick="window.handleInventoryAction('${bot.id}', ${item.id}, 'TRASH')">Trash</button>`;


            row.innerHTML = `
                <td class="inv-info-cell">
                    <span class="inv-count">${item.count}x</span>
                    <span class="inv-sep">|</span>
                    <span class="inv-name">${name} [${item.id}]</span>
                    ${item.is_active ? '<span class="inv-equipped">(Equipped)</span>' : ''}
                </td>
                <td class="inv-action-cell">
                    ${buttonsHtml}
                </td>
            `;
            bodyEl.appendChild(row);
        });
    }

    window.handleInventoryAction = (botId, itemId, action) => {
        if (!socket || socket.readyState !== WebSocket.OPEN) return;

        console.log(`[Inventory] Action ${action} for Item ${itemId} on Bot ${botId}`);
        socket.send(JSON.stringify({
            type: 'BOT_ACTION',
            data: {
                id: botId,
                action: 'INVENTORY_ACTION',
                sub_action: action,
                item_id: itemId
            }
        }));
    };

    // --- Event Listeners (Once) ---
    document.querySelectorAll('.nav-item').forEach(item => {
        item.onclick = () => {
            document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
            document.querySelectorAll('.view-section').forEach(v => v.classList.remove('active'));
            item.classList.add('active');
            const target = item.dataset.target;
            document.getElementById(target).classList.add('active');

            if (target === 'database-view') {
                socket.send(JSON.stringify({ type: 'GET_DATABASE_INFO' }));
            }
        };
    });

    document.getElementById('remove-bot-btn').onclick = () => {
        if (!selectedBotId) return;
        const idToRemove = selectedBotId;
        socket.send(JSON.stringify({ type: 'REMOVE_BOT', data: { id: idToRemove } }));
        delete botDebugLogs[idToRemove];
        deselectBot();
    };

    document.getElementById('btn-connect').onclick = () => {
        if (selectedBotId) socket.send(JSON.stringify({ type: 'BOT_ACTION', data: { id: selectedBotId, action: 'CONNECT' } }));
    };

    document.getElementById('btn-disconnect').onclick = () => {
        if (selectedBotId) socket.send(JSON.stringify({ type: 'BOT_ACTION', data: { id: selectedBotId, action: 'DISCONNECT' } }));
    };

    document.getElementById('btn-leave').onclick = () => {
        if (selectedBotId) socket.send(JSON.stringify({ type: 'BOT_ACTION', data: { id: selectedBotId, action: 'LEAVE' } }));
    };

    document.getElementById('btn-warp').onclick = () => {
        const world = document.getElementById('world-input').value;
        if (selectedBotId && world) {
            socket.send(JSON.stringify({ type: 'BOT_ACTION', data: { id: selectedBotId, action: 'WARP', world: world } }));
        }
    };

    document.getElementById('btn-say').onclick = () => {
        const text = document.getElementById('say-input').value;
        if (selectedBotId && text) {
            socket.send(JSON.stringify({ type: 'BOT_ACTION', data: { id: selectedBotId, action: 'SAY', text: text } }));
            document.getElementById('say-input').value = '';
        }
    };

    document.getElementById('detail-glog').oninput = () => debounceSave();
    document.getElementById('detail-proxy').oninput = () => debounceSave();
    document.getElementById('detail-glog').onblur = () => { if (saveTimeout) clearTimeout(saveTimeout); saveBotConfig(); };
    document.getElementById('detail-proxy').onblur = () => { if (saveTimeout) clearTimeout(saveTimeout); saveBotConfig(); };

    function debounceSave() {
        if (saveTimeout) clearTimeout(saveTimeout);
        saveTimeout = setTimeout(saveBotConfig, 500);
    }

    function saveBotConfig() {
        if (!selectedBotId) return;
        const glog = document.getElementById('detail-glog').value;
        const proxy = document.getElementById('detail-proxy').value;
        socket.send(JSON.stringify({ type: 'UPDATE_BOT_CONFIG', data: { id: selectedBotId, glog, proxy } }));
    }

    document.getElementById('btn-execute-lua').onclick = () => {
        const script = document.getElementById('lua-editor').value;
        if (selectedBotId && script) {
            socket.send(JSON.stringify({ type: 'EXECUTE_LUA', data: { id: selectedBotId, script: script } }));
            const consoleBox = document.getElementById('bot-console');
            const entry = document.createElement('div');
            entry.className = 'log-line';
            entry.textContent = `> Sent script to ${selectedBotId}`;
            consoleBox.appendChild(entry);
            consoleBox.scrollTop = consoleBox.scrollHeight;
        }
    };

    // Modal
    const modal = document.getElementById('add-bot-modal');
    document.getElementById('add-bot-btn').onclick = () => {
        document.querySelector('input[value="legacy"]').checked = true;
        document.querySelectorAll('.radio-card').forEach(card => card.classList.remove('active'));
        document.querySelector('input[value="legacy"]').closest('.radio-card').classList.add('active');
        document.getElementById('field-legacy-name').classList.remove('hidden');
        document.getElementById('field-legacy-pass').classList.remove('hidden');
        document.getElementById('field-token-combined').classList.add('hidden');
        document.getElementById('field-external-password').classList.add('hidden');
        document.getElementById('field-glog').classList.add('hidden');
        modal.classList.add('active');
    };
    document.querySelectorAll('.close-modal').forEach(el => el.onclick = () => modal.classList.remove('active'));

    document.querySelectorAll('input[name="bot-type"]').forEach(radio => {
        radio.onchange = (e) => {
            const type = e.target.value;
            document.querySelectorAll('.radio-card').forEach(card => card.classList.remove('active'));
            e.target.closest('.radio-card').classList.add('active');
            if (type === 'legacy') {
                document.getElementById('field-legacy-name').classList.remove('hidden');
                document.getElementById('field-legacy-pass').classList.remove('hidden');
                document.getElementById('field-token-combined').classList.add('hidden');
                document.getElementById('field-external-password').classList.add('hidden');
                document.getElementById('field-glog').classList.add('hidden');
            } else {
                document.getElementById('field-legacy-name').classList.add('hidden');
                document.getElementById('field-legacy-pass').classList.add('hidden');
                document.getElementById('field-token-combined').classList.remove('hidden');
                document.getElementById('field-external-password').classList.remove('hidden');
                document.getElementById('field-glog').classList.remove('hidden');
            }
        };
    });

    document.getElementById('confirm-add-bot').onclick = () => {
        const type = document.querySelector('input[name="bot-type"]:checked').value;
        let name = "", pass = "", glog = document.getElementById('bot-glog').value;
        const proxy = document.getElementById('bot-proxy-input').value;

        if (type === 'legacy') {
            name = document.getElementById('bot-identity').value;
            pass = document.getElementById('bot-password').value;
        } else {
            const combined = document.getElementById('bot-token-combined').value.trim();
            const externalPass = document.getElementById('bot-external-password').value.trim();
            name = combined.split('|')[0] || type;
            pass = combined;
            // Append external password if provided
            if (externalPass) {
                pass = pass + '|' + externalPass;
            }
        }

        if (!name && !pass) { alert('Mohon isi data bot.'); return; }
        socket.send(JSON.stringify({ type: 'ADD_BOT', data: { type, name, pass, glog, proxy } }));
        modal.classList.remove('active');
        document.getElementById('bot-identity').value = '';
        document.getElementById('bot-password').value = '';
        document.getElementById('bot-token-combined').value = '';
        document.getElementById('bot-external-password').value = '';
        document.getElementById('bot-glog').value = '';
        document.getElementById('bot-proxy-input').value = '';
    };

    document.querySelectorAll('.tab-link').forEach(link => {
        link.onclick = () => {
            document.querySelectorAll('.tab-link').forEach(l => l.classList.remove('active'));
            document.querySelectorAll('.tab-pane').forEach(p => p.classList.remove('active'));
            link.classList.add('active');
            document.getElementById(link.dataset.tab).classList.add('active');
        };
    });

    // --- Debug Helpers ---
    const debugFilterAll = document.getElementById('debug-all');
    const debugFilterHttps = document.getElementById('debug-https');
    const debugAutoScroll = document.getElementById('debug-autoscroll');
    const debugLimitInput = document.getElementById('debug-limit');

    function appendDebugLog(log) {
        if (!botDebugLogs[log.bot_id]) botDebugLogs[log.bot_id] = [];
        const entry = document.createElement('div');
        entry.className = `log-entry ${log.is_error ? 'error' : ''} ${log.category.toLowerCase() === 'https' ? 'https-in' : ''}`;
        if (log.message.includes('POST') || log.message.includes('GET')) {
            entry.classList.remove('https-in');
            entry.classList.add('https-out');
        }
        entry.innerHTML = `<span style="color: #64748b; margin-right: 8px;">[${log.time}]</span><span class="debug-tag">${log.category}</span> ${escapeHtml(log.message)}`;

        botDebugLogs[log.bot_id].push(entry.outerHTML);

        const limit = parseInt(debugLimitInput.value) || 100;
        if (botDebugLogs[log.bot_id].length > limit) {
            botDebugLogs[log.bot_id] = botDebugLogs[log.bot_id].slice(-limit);
        }

        if (selectedBotId === log.bot_id) {
            if (log.category === 'HTTPS' && !debugFilterHttps.checked && !debugFilterAll.checked) return;
            debugLogsContainer.appendChild(entry);

            // Limit DOM entries too for performance
            while (debugLogsContainer.children.length > limit) {
                debugLogsContainer.removeChild(debugLogsContainer.firstChild);
            }

            if (debugAutoScroll.checked) {
                debugLogsContainer.scrollTop = debugLogsContainer.scrollHeight;
            }
        }
    }

    function refreshDebugView(botId) {
        debugLogsContainer.innerHTML = '';
        if (!botDebugLogs[botId] || botDebugLogs[botId].length === 0) {
            debugLogsContainer.innerHTML = '<div class="log-entry system">No logs for this bot.</div>';
            return;
        }

        const limit = parseInt(debugLimitInput.value) || 100;
        const logsToShow = botDebugLogs[botId].slice(-limit);

        logsToShow.forEach(html => {
            const temp = document.createElement('div');
            temp.innerHTML = html;
            const entry = temp.firstChild;
            const tag = entry.querySelector('.debug-tag')?.textContent;
            if (tag === 'HTTPS' && !debugFilterHttps.checked && !debugFilterAll.checked) return;
            debugLogsContainer.appendChild(entry);
        });

        if (debugAutoScroll.checked) {
            debugLogsContainer.scrollTop = debugLogsContainer.scrollHeight;
        }
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML.replace(/\n/g, '<br>');
    }

    document.getElementById('clear-debug').onclick = () => {
        if (selectedBotId) { botDebugLogs[selectedBotId] = []; refreshDebugView(selectedBotId); }
    };

    [debugFilterAll, debugFilterHttps, debugLimitInput].forEach(el => {
        el.onchange = () => { if (selectedBotId) refreshDebugView(selectedBotId); };
    });

    document.getElementById('debug-enet').onchange = (e) => {
        if (selectedBotId) {
            socket.send(JSON.stringify({
                type: 'UPDATE_BOT_CONFIG',
                data: { id: selectedBotId, show_enet: e.target.checked }
            }));
        }
    };

    // --- Database View Implementation ---
    const dbSearchInput = document.getElementById('db-search-input');
    const dbItemList = document.getElementById('db-item-list');
    const dbItemCount = document.getElementById('db-item-count');
    const itemInfoContent = document.getElementById('item-info-content');
    const dbNoSelection = document.querySelector('#item-detail-card .no-selection');

    let dbSearchTimeout = null;

    if (dbSearchInput) {
        dbSearchInput.oninput = () => {
            if (dbSearchTimeout) clearTimeout(dbSearchTimeout);
            const query = dbSearchInput.value.trim();
            if (query.length < 2) {
                dbItemList.innerHTML = '<div class="empty-state">Start typing to search items</div>';
                dbItemCount.textContent = '0';
                return;
            }
            dbSearchTimeout = setTimeout(() => {
                socket.send(JSON.stringify({ type: 'SEARCH_ITEMS', data: { query } }));
            }, 300);
        };
    }

    function renderDBItemList(items) {
        if (!dbItemList) return;
        dbItemList.innerHTML = '';
        dbItemCount.textContent = items ? items.length : 0;

        if (!items || items.length === 0) {
            dbItemList.innerHTML = '<div class="empty-state">No items found</div>';
            return;
        }

        items.forEach(item => {
            const el = document.createElement('div');
            el.className = 'db-item';
            el.innerHTML = `
                <div class="item-info">
                    <span class="item-name">${item.Name}</span>
                    <span class="item-id">ID: ${item.ID}</span>
                </div>
                <i class="fa-solid fa-chevron-right" style="font-size: 0.8rem; opacity: 0.5;"></i>
            `;
            el.onclick = () => {
                document.querySelectorAll('.db-item').forEach(i => i.classList.remove('active'));
                el.classList.add('active');

                const cached = window.getItem(item.ID);
                if (cached) {
                    renderItemDetail(cached);
                } else {
                    // This fallback should rarely happen if search results just populated cache
                    socket.send(JSON.stringify({ type: 'GET_ITEM', data: { id: item.ID } }));
                }
            };
            dbItemList.appendChild(el);
        });
    }

    function renderItemDetail(item) {
        if (!item || !itemInfoContent) return;

        if (dbNoSelection) dbNoSelection.classList.add('hidden');
        itemInfoContent.classList.remove('hidden');

        // Reset scroll position
        const rightPanel = document.querySelector('.db-right-panel');
        if (rightPanel) {
            rightPanel.scrollTop = 0;
        }

        // General Info
        document.getElementById('item-name').textContent = item.Name || "Unknown Item";
        document.getElementById('item-id').textContent = `ID: ${item.ID}`;
        document.getElementById('item-rarity').textContent = item.Rarity || 0;

        let growTime = item.GrowTime || 0;
        let growTimeStr = "";
        if (growTime === 0) growTimeStr = "Instant";
        else {
            const h = Math.floor(growTime / 3600);
            const m = Math.floor((growTime % 3600) / 60);
            const s = growTime % 60;
            if (h > 0) growTimeStr += `${h}h `;
            if (m > 0) growTimeStr += `${m}m `;
            if (s > 0) growTimeStr += `${s}s`;
        }
        document.getElementById('item-grow-time').textContent = growTimeStr || "0s";
        document.getElementById('item-description').textContent = item.Description || "No description available.";

        // Gameplay Section
        document.getElementById('item-action-type').textContent = item.ActionType || 0;
        document.getElementById('item-material').textContent = item.Material || 0;
        document.getElementById('item-collision-type').textContent = item.CollisionType || 0;
        document.getElementById('item-clothing-type').textContent = item.ClothingType || 0;
        document.getElementById('item-health').textContent = item.BlockHealth || 0;
        document.getElementById('item-max-item').textContent = item.MaxItem || 0;
        document.getElementById('item-drop-chance').textContent = item.DropChance || 0;
        document.getElementById('item-rayman').textContent = item.IsRayman === 1 ? "Yes" : "No";

        // Visual Section
        document.getElementById('item-texture').textContent = item.TextureFileName || "-";
        document.getElementById('item-texture-hash').textContent = (item.TextureHash || 0).toString(16).toUpperCase();
        document.getElementById('item-texture-pos').textContent = `${item.TextureX || 0}, ${item.TextureY || 0}`;
        document.getElementById('item-render-type').textContent = item.RenderType || 0;
        document.getElementById('item-visual-effect').textContent = item.VisualEffect || 0;
        document.getElementById('item-file-name').textContent = item.FileName || "-";
        document.getElementById('item-file-hash').textContent = (item.FileHash || 0).toString(16).toUpperCase();

        // Seed/Tree Section
        document.getElementById('item-seed-sprites').textContent = `${item.SeedBaseSprite || 0} / ${item.SeedOverlaySprite || 0}`;
        document.getElementById('item-tree-sprites').textContent = `${item.TreeBaseSprite || 0} / ${item.TreeOverlaySprite || 0}`;

        const baseColor = document.getElementById('item-base-color');
        const overlayColor = document.getElementById('item-overlay-color');

        const hexBase = '#' + (item.BaseColor || 0).toString(16).padStart(8, '0').substring(2);
        const hexOverlay = '#' + (item.OverlayColor || 0).toString(16).padStart(8, '0').substring(2);

        baseColor.textContent = hexBase.toUpperCase();
        baseColor.style.color = hexBase;

        overlayColor.textContent = hexOverlay.toUpperCase();
        overlayColor.style.color = hexOverlay;

        document.getElementById('item-ingredient').textContent = item.Ingredient || 0;
        document.getElementById('item-cooking-ing').textContent = item.CookingIngredient || 0;

        // Pet Section
        document.getElementById('item-pet-name').textContent = item.PetName || "-";
        document.getElementById('item-pet-prefix').textContent = item.PetPrefix || "-";
        document.getElementById('item-pet-suffix').textContent = item.PetSuffix || "-";
        document.getElementById('item-pet-ability').textContent = item.PetAbility || "-";

        // Extra Section
        document.getElementById('item-punch-option').textContent = item.PunchOption || "-";
        document.getElementById('item-stripey').textContent = item.IsStripeyWallpaper === 1 ? "Yes" : "No";
        document.getElementById('item-audio').textContent = item.AudioVolume || 0;
        document.getElementById('item-texture2').textContent = item.TexturePath2 || "-";
        document.getElementById('item-extra-opt').textContent = item.ExtraOptions || "-";
        document.getElementById('item-extra-opt2').textContent = item.ExtraOption2 || "-";

        // Render Flags
        const flagsContainer = document.getElementById('item-flags');
        if (flagsContainer) {
            flagsContainer.innerHTML = '';
            if (item.Flags) {
                Object.entries(item.Flags).forEach(([key, value]) => {
                    const flagEl = document.createElement('div');
                    flagEl.className = `flag-item ${value ? 'enabled' : 'disabled'}`;
                    flagEl.innerHTML = `
                        <i class="fa-solid ${value ? 'fa-check-circle' : 'fa-circle-xmark'}"></i>
                        ${key}
                    `;
                    flagsContainer.appendChild(flagEl);
                });
            }
        }
    }

    function renderDatabaseInfo(info) {
        if (!info) return;
        const statusBadge = document.getElementById('db-status-badge');
        if (statusBadge) {
            statusBadge.textContent = info.loaded ? 'Loaded' : 'Error';
            statusBadge.className = info.loaded ? 'badge success' : 'badge danger';
        }
        const versionEl = document.getElementById('db-version');
        if (versionEl) versionEl.textContent = info.version || 0;

        const countEl = document.getElementById('db-total-items');
        if (countEl) countEl.textContent = (info.item_count || 0).toLocaleString();
    }

    // ===== WORLD / MAP RENDERING =====
    let currentZoom = 1;
    let panX = 0;
    let panY = 0;
    let isDragging = false;
    let dragStartX = 0;
    let dragStartY = 0;

    // Interactive State
    let activeWorld = null;
    let hoveredTile = null;
    const TOOLTIP_ID = 'map-tooltip';

    // Global Constants for Rendering
    const TILE_WIDTH = 8.5;
    const TILE_HEIGHT = 6;

    // Create Tooltip Element if not exists
    let mapTooltip = document.getElementById(TOOLTIP_ID);
    if (!mapTooltip) {
        mapTooltip = document.createElement('div');
        mapTooltip.id = TOOLTIP_ID;
        mapTooltip.className = 'map-floating-tooltip hidden';
        document.body.appendChild(mapTooltip);
    }

    function renderWorldMap(bot) {
        if (!bot || !bot.local || !bot.local.world) return;

        const world = bot.local.world;
        if (!world.width || !world.height) return;

        // Update active world reference
        activeWorld = world;

        // Update UI info
        document.getElementById('world-name').textContent = world.name || 'EXIT';
        document.getElementById('world-dimensions').textContent = `${world.width}x${world.height}`;
        document.getElementById('world-tiles').textContent = `${world.tile_count || 0} tiles`;

        drawMapCanvas();
    }

    function drawMapCanvas() {
        const canvas = document.getElementById('world-canvas');
        if (!canvas || !activeWorld) return;

        const ctx = canvas.getContext('2d');

        // Logical Size
        const logicWidth = activeWorld.width * TILE_WIDTH;
        const logicHeight = activeWorld.height * TILE_HEIGHT;

        // Set Display Size (Canvas Resolution)
        // We keep the internal resolution high enough for the zoom
        canvas.width = logicWidth;
        canvas.height = logicHeight;

        // CSS handling is done via applyCanvasZoom transformation
        // Clear
        ctx.fillStyle = '#84C5E2'; // Sky Blue
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        // --- Render Tiles ---
        if (activeWorld.tiles) {
            activeWorld.tiles.forEach(tile => {
                const x = (tile.x || 0) * TILE_WIDTH;
                const y = (tile.y || 0) * TILE_HEIGHT;

                // 1. Background
                if (tile.bg_id > 0) {
                    const col = getTileColor(tile.bg_id);
                    if (col) {
                        ctx.fillStyle = col;
                        ctx.fillRect(x, y, TILE_WIDTH, TILE_HEIGHT);
                        // Darken BG
                        ctx.fillStyle = 'rgba(0, 0, 0, 0.3)';
                        ctx.fillRect(x, y, TILE_WIDTH, TILE_HEIGHT);
                    }
                }

                // 2. Foreground
                if (tile.fg_id > 0) {
                    const col = getTileColor(tile.fg_id);
                    if (col) {
                        ctx.fillStyle = col;
                        ctx.fillRect(x, y, TILE_WIDTH, TILE_HEIGHT);
                    } else {
                        // Default Unknown / Generic Block
                        ctx.fillStyle = '#FF00FF'; // Debug Magenta for unknown
                        // ctx.fillRect(x, y, TILE_WIDTH, TILE_HEIGHT);
                    }
                }
            });
        }

        // --- Render Hover Highlight ---
        if (hoveredTile) {
            const hx = hoveredTile.x * TILE_WIDTH;
            const hy = hoveredTile.y * TILE_HEIGHT;

            // Brighten effect
            ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
            ctx.fillRect(hx, hy, TILE_WIDTH, TILE_HEIGHT);

            // Border
            ctx.strokeStyle = '#FFFFFF';
            ctx.lineWidth = 1;
            ctx.strokeRect(hx - 0.5, hy - 0.5, TILE_WIDTH + 1, TILE_HEIGHT + 1);
        }

        applyCanvasZoom();
    }

    // Manual Tile Color Mapping
    const TILE_COLORS = {
        0: '#84C5E2',   // Empty (Sky)
        2: '#8B4513',  // Dirt (Brown)
        3: '#8B4513', // Dirt Seed
        4: '#fd8607',   // Lava
        5: '#fd8607',   // Lava Seed
        6: '#8a2be2',  // Main Door
        8: '#222222',  // Bedrock
        10: '#b9c2c6', // Rock
        11: '#b9c2c6', // Rock Seed
        12: '#864801', // Door
        13: '#864801', // Door Seed
        14: '#5e401b', // Cave background
        15: '#5e401b', // Cave background Seed
        20: '#efdec4', // Sign
        22: '#654321', // Sign background (approx)
        24: '#654321', // Super Sign
        340: '#debc22', // Chandelier
        341: '#debc22', // Chandelier Seed
        5666: '#ff0000', // Laser grid
        5667: '#ff0000', // Laser grid Seed
        4584: '#00ff26', // Pepper tree
        4585: '#00ff26', // Pepper tree Seed
    };

    function getTileColor(itemId) {
        // 1. Check manual map
        if (TILE_COLORS.hasOwnProperty(itemId)) {
            return TILE_COLORS[itemId];
        }

        // 2. Check Item Cache (from Database)
        const item = window.getItem(itemId);
        if (item) {
            if (item.BaseColor) {
                const unsignedColor = item.BaseColor >>> 0;
                let hexC = unsignedColor.toString(16).padStart(8, '0');
                // Format usually AARRGGBB in memory, but HTML needs #RRGGBB
                // Actually in GT it might be AABBGGRR or similar. Assuming ARGB or RGBA.
                // Let's try standard interpretation.
                // If it looks wrong, we fix it.
                // Usually: 0xFFRRGGBB
                const r = hexC.substring(2, 4);
                const g = hexC.substring(4, 6);
                const b = hexC.substring(6, 8);
                return `#${r}${g}${b}`;
            }
        }

        // 3. Fallback
        return null;
    }

    function applyCanvasZoom() {
        const canvas = document.getElementById('world-canvas');
        if (!canvas) return;

        canvas.style.transform = `translate(${panX}px, ${panY}px) scale(${currentZoom})`;
        canvas.style.transformOrigin = 'top left';

        document.getElementById('zoom-level').textContent = Math.round(currentZoom * 100) + '%';
    }

    // Zoom controls
    document.getElementById('zoom-in')?.addEventListener('click', () => {
        currentZoom = Math.min(currentZoom * 1.2, 5);
        applyCanvasZoom();
    });

    document.getElementById('zoom-out')?.addEventListener('click', () => {
        currentZoom = Math.max(currentZoom / 1.2, 0.5);
        applyCanvasZoom();
    });

    document.getElementById('reset-view')?.addEventListener('click', () => {
        currentZoom = 1;
        panX = 0;
        panY = 0;
        applyCanvasZoom();
    });

    // Pan & Hover controls
    const canvasContainer = document.querySelector('.canvas-container');

    if (canvasContainer) {
        canvasContainer.addEventListener('mousedown', (e) => {
            if (e.button === 0) { // Left click to drag
                isDragging = true;
                dragStartX = e.clientX - panX;
                dragStartY = e.clientY - panY;
                canvasContainer.style.cursor = 'grabbing';
            }
        });

        canvasContainer.addEventListener('mousemove', (e) => {
            // Panning
            if (isDragging) {
                panX = e.clientX - dragStartX;
                panY = e.clientY - dragStartY;
                applyCanvasZoom();
                return; // Don't process grid hover while dragging
            }

            // Hover Logic
            if (!activeWorld) return;

            const rect = canvasContainer.getBoundingClientRect();

            // Calculate mouse position relative to the container/canvas
            // Taking zoom and pan into account is tricky because 'transform' is CSS only.
            // The mouse coordinates are relative to the viewport.
            // visualX = mouseX - rect.left
            // logicX = (visualX - panX) / currentZoom

            const mouseX = e.clientX - rect.left;
            const mouseY = e.clientY - rect.top;

            const logicX = (mouseX - panX) / currentZoom;
            const logicY = (mouseY - panY) / currentZoom;

            // Convert Logic coords to Tile coords
            const tileX = Math.floor(logicX / TILE_WIDTH);
            const tileY = Math.floor(logicY / TILE_HEIGHT);

            // Check Bounds
            if (tileX >= 0 && tileX < activeWorld.width && tileY >= 0 && tileY < activeWorld.height) {
                // Optimized lookup: Tiles are sent in row-major order
                const idx = tileX + tileY * activeWorld.width;
                let tile = activeWorld.tiles[idx];

                // Extra safety check in case of sparse or reordered array
                if (!tile || (tile.x !== tileX || tile.y !== tileY)) {
                    tile = activeWorld.tiles.find(t => t.x === tileX && t.y === tileY);
                }

                if (tile !== hoveredTile) {
                    hoveredTile = tile; // Can be undefined if empty/air
                    // Create minimal dummy tile if undefined (air)
                    if (!hoveredTile) {
                        hoveredTile = { x: tileX, y: tileY, fg_id: 0, bg_id: 0 };
                    }
                    drawMapCanvas(); // Re-render to show highlight
                }

                // Update Tooltip
                updateTooltip(e, hoveredTile);
            } else {
                if (hoveredTile) {
                    hoveredTile = null;
                    drawMapCanvas();
                    hideTooltip();
                }
            }
        });

        canvasContainer.addEventListener('mouseup', () => {
            isDragging = false;
            canvasContainer.style.cursor = 'default';
        });

        canvasContainer.addEventListener('mouseleave', () => {
            isDragging = false;
            hoveredTile = null;
            drawMapCanvas();
            hideTooltip();
        });
    }

    function updateTooltip(e, tile) {
        if (!tile) return;

        const fgItem = window.getItem(tile.fg_id);
        const bgItem = window.getItem(tile.bg_id);

        const fgName = fgItem ? fgItem.Name : (tile.fg_id === 0 ? "Empty" : `ID: ${tile.fg_id}`);
        const bgName = bgItem ? bgItem.Name : (tile.bg_id === 0 ? "Empty" : `ID: ${tile.bg_id}`);

        let html = `
            <div class="tooltip-header"><i class="fa-solid fa-location-dot"></i> (${tile.x}, ${tile.y})</div>
            <div class="tooltip-row"><span class="label">FG:</span> <span class="val">${fgName}</span></div>
            <div class="tooltip-row"><span class="label">BG:</span> <span class="val">${bgName}</span></div>
        `;

        // Add Flags info or generic info
        if (tile.flags) {
            html += `<div class="tooltip-row"><span class="label">Flags:</span> <span class="val dimmed">${tile.flags}</span></div>`;
        }

        mapTooltip.innerHTML = html;
        mapTooltip.classList.remove('hidden');

        // Position info
        const offset = 15;
        // Keep within viewport
        let top = e.clientY + offset;
        let left = e.clientX + offset;

        // Simple bounds check could go here

        mapTooltip.style.top = `${top}px`;
        mapTooltip.style.left = `${left}px`;
    }

    function hideTooltip() {
        mapTooltip.classList.add('hidden');
    }

    // World tab switching
    document.querySelectorAll('.world-tab-link').forEach(link => {
        link.addEventListener('click', () => {
            const targetTab = link.getAttribute('data-world-tab');

            // Update active states
            document.querySelectorAll('.world-tab-link').forEach(l => l.classList.remove('active'));
            document.querySelectorAll('.world-tab-pane').forEach(p => p.classList.remove('active'));

            link.classList.add('active');
            document.getElementById(targetTab)?.classList.add('active');
        });
    });

    // Growscan functionality
    function updateGrowscan(bot) {
        if (!bot || !bot.local || !bot.local.world) return;

        const world = bot.local.world;
        const growscanList = document.getElementById('growscan-list');
        const growscanCount = document.getElementById('growscan-count');

        if (!growscanList || !world.tiles) return;

        // Find harvestable tiles (seeds with time passed)
        const harvestable = world.tiles.filter(tile => {
            if (tile.fg_id === 0) return false;

            const item = window.getItem(tile.fg_id);
            if (!item) return false;

            // Check if it's a seed and has grow time
            if (tile.tile_type === 4 && tile.extra) {
                if (tile.extra.ready_to_harvest) return true;
                if (item.GrowTime > 0 && tile.extra.time_passed >= item.GrowTime) return true;
            }
            return false;
        });

        growscanCount.textContent = harvestable.length;

        if (harvestable.length === 0) {
            // ... (rest is same)
            growscanList.innerHTML = `
                <div style="padding: 40px; text-align: center; opacity: 0.5;">
                    <i class="fa-solid fa-seedling" style="font-size: 48px; margin-bottom: 20px;"></i>
                    <p>No harvestable tiles found</p>
                </div>
            `;
            return;
        }

        // Render harvestable tiles
        growscanList.innerHTML = harvestable.map(tile => {
            const item = window.getItem(tile.fg_id);
            const itemName = item ? item.Name : `Item ${tile.fg_id}`;

            return `
                <div class="growscan-item">
                    <div class="growscan-item-info">
                        <div class="growscan-item-icon">
                            <i class="fa-solid fa-seedling"></i>
                        </div>
                        <div class="growscan-item-details">
                            <h5>${itemName}</h5>
                            <p>Position: (${tile.x}, ${tile.y})</p>
                        </div>
                    </div>
                    <div class="growscan-item-actions">
                        <button class="btn btn-sm success" onclick="harvestTile(${tile.x}, ${tile.y})">
                            <i class="fa-solid fa-hand-sparkles"></i> Harvest
                        </button>
                    </div>
                </div>
            `;
        }).join('');
    }

    // Global function for harvest action
    window.harvestTile = function (x, y) {
        if (!selectedBotId) return;

        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({
                type: 'BOT_ACTION',
                bot_id: selectedBotId,
                action: 'HARVEST_TILE',
                data: { x, y }
            }));
        }
    };
});
