/* PLC Gateway – front-end (vanilla JS, sem dependências externas) */
'use strict';

const LOG = '[PLC-Gateway]';
const $ = (id) => document.getElementById(id);

const state = {
    devices: [],
    tags: [],
    live: {},          // id da tag -> {value, quality, timestamp}
    statuses: {},      // id do device -> {state, message}
    net: null,         // resposta de /api/network/interfaces
    page: 'dashboard',
    showVirtual: false,
};

/* ---------- utilidades ---------- */
function esc(v) {
    return String(v ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

function toast(msg, type = 'info') {
    const el = document.createElement('div');
    el.className = `toast ${type}`;
    el.textContent = msg;
    $('toasts').appendChild(el);
    setTimeout(() => el.remove(), type === 'error' ? 6000 : 3000);
}

async function api(method, url, body) {
    console.log(`${LOG} ${method} ${url}`, body ?? '');
    let res;
    try {
        res = await fetch(url, {
            method,
            headers: body ? { 'Content-Type': 'application/json' } : undefined,
            body: body ? JSON.stringify(body) : undefined,
        });
    } catch (e) {
        console.error(`${LOG} falha de rede em ${method} ${url}`, e);
        throw new Error('Sem comunicação com o servidor');
    }
    let data = null;
    try { data = await res.json(); } catch (_) { /* corpo vazio */ }
    if (!res.ok) {
        const msg = (data && data.erro) || `Erro ${res.status}`;
        console.error(`${LOG} ${method} ${url} -> ${res.status}: ${msg}`);
        throw new Error(msg);
    }
    console.log(`${LOG} ${method} ${url} -> ${res.status}`, data);
    return data;
}

function confirmDialog(title, text, okLabel = 'Confirmar') {
    return new Promise((resolve) => {
        const dlg = $('dlgConfirm');
        $('cfTitle').textContent = title;
        $('cfText').textContent = text;
        $('cfOk').textContent = okLabel;
        dlg.returnValue = 'cancel';
        dlg.addEventListener('close', () => resolve(dlg.returnValue === 'ok'), { once: true });
        dlg.showModal();
    });
}

const deviceById = (id) => state.devices.find((d) => d.id === id);

function fmtValue(tag) {
    const l = state.live[tag.id];
    if (!l) return { text: '—', cls: '', bad: false };
    if (!l.quality) return { text: 'ERRO', cls: '', bad: true };
    const v = l.value;
    if (tag.data_type === 'BOOL') {
        const t = v === true || v === 1;
        return { text: t ? 'TRUE' : 'FALSE', cls: t ? 'true' : 'false', bad: false };
    }
    if (typeof v === 'number') return { text: Number.isInteger(v) ? String(v) : v.toFixed(2), cls: '', bad: false };
    return { text: v == null ? '—' : String(v), cls: '', bad: false };
}

const STATE_LABEL = { online: 'Online', offline: 'Offline', connecting: 'A conectar', unknown: 'Desconhecido' };
function stateBadge(deviceId) {
    const st = state.statuses[deviceId] || { state: 'unknown', message: '' };
    const s = STATE_LABEL[st.state] ? st.state : 'unknown';
    return `<span class="state" title="${esc(st.message)}"><span class="dot dot-${s}"></span><span class="badge ${s}">${STATE_LABEL[s]}</span></span>`;
}

/* ---------- carregamento de dados ---------- */
async function loadConfig() {
    const cfg = await api('GET', '/api/config');
    state.devices = cfg.devices || [];
    state.tags = cfg.tags || [];
    fillDeviceFilters();
    render();
}

async function loadStatuses() {
    try {
        state.statuses = await api('GET', '/api/devices/status');
        if (state.page === 'devices' || state.page === 'dashboard') render();
    } catch (e) { /* já logado */ }
}

async function loadNetwork() {
    try {
        state.net = await api('GET', '/api/network/interfaces');
        if (state.page === 'network') renderNetwork();
    } catch (e) {
        $('netBanner').innerHTML = `<div class="banner">Erro ao ler interfaces: ${esc(e.message)}</div>`;
    }
}

function fillDeviceFilters() {
    for (const id of ['dashDeviceFilter', 'tagDeviceFilter']) {
        const sel = $(id);
        const cur = sel.value;
        sel.innerHTML = '<option value="">Todos os dispositivos</option>' +
            state.devices.map((d) => `<option value="${esc(d.id)}">${esc(d.name)} (${esc(d.id)})</option>`).join('');
        sel.value = state.devices.some((d) => d.id === cur) ? cur : '';
    }
}

/* ---------- renderização ---------- */
let renderPending = false;
function render() {
    if (renderPending) return;
    renderPending = true;
    requestAnimationFrame(() => {
        renderPending = false;
        if (state.page === 'dashboard') renderDashboard();
        else if (state.page === 'devices') renderDevices();
        else if (state.page === 'tags') renderTags();
        else if (state.page === 'network') renderNetwork();
    });
}

function renderDashboard() {
    const filter = $('dashDeviceFilter').value;
    const devices = state.devices.filter((d) => !filter || d.id === filter);
    const online = state.devices.filter((d) => (state.statuses[d.id] || {}).state === 'online').length;
    const bad = state.tags.filter((t) => state.live[t.id] && !state.live[t.id].quality).length;
    $('dashSummary').innerHTML =
        `<div class="stat"><b>${online}/${state.devices.length}</b><span>Dispositivos online</span></div>
         <div class="stat"><b>${state.tags.length}</b><span>Tags configuradas</span></div>
         <div class="stat"><b>${bad}</b><span>Tags com erro</span></div>`;

    if (!state.devices.length) {
        $('dashGroups').innerHTML = '<div class="panel"><p class="empty">Nenhum dispositivo cadastrado. Vá em <a href="#devices" style="color:var(--accent)">Dispositivos</a> para começar.</p></div>';
        return;
    }
    $('dashGroups').innerHTML = devices.map((d) => {
        const tags = state.tags.filter((t) => t.device_id === d.id);
        const cards = tags.map((t) => {
            const v = fmtValue(t);
            const l = state.live[t.id];
            const time = l && l.timestamp ? new Date(l.timestamp).toLocaleTimeString() : '--:--:--';
            return `<div class="card ${v.bad ? 'bad' : ''}">
                <div class="name"><span>${esc(t.name)}</span><span class="badge">${esc(t.data_type)}</span></div>
                <div class="val ${v.cls}">${esc(v.text)}</div>
                <div class="meta">${esc(t.id)} · ${time}</div></div>`;
        }).join('') || '<p class="sub">Sem tags neste dispositivo.</p>';
        return `<div class="group"><h3>${stateBadge(d.id)} ${esc(d.name)} <span class="sub">${esc(d.protocol.toUpperCase())} · ${esc(d.ip_address)}</span></h3>
            <div class="cards">${cards}</div></div>`;
    }).join('');
}

function renderDevices() {
    const body = $('devicesBody');
    if (!state.devices.length) {
        body.innerHTML = '<tr><td colspan="8" class="empty">Nenhum dispositivo cadastrado</td></tr>';
        return;
    }
    body.innerHTML = state.devices.map((d) => {
        const st = state.statuses[d.id] || {};
        const count = state.tags.filter((t) => t.device_id === d.id).length;
        return `<tr>
            <td>${stateBadge(d.id)}<div class="sub">${esc(st.message || '')}</div></td>
            <td><code>${esc(d.id)}</code></td><td>${esc(d.name)}</td>
            <td>${d.protocol === 's7' ? 'Siemens S7' : 'Rockwell CIP'}</td>
            <td>${esc(d.ip_address)}</td>
            <td>${d.protocol === 's7' ? `${d.rack}/${d.slot}` : '—'}</td>
            <td>${count}</td>
            <td class="right">
                <button class="btn sm" data-action="device-edit" data-id="${esc(d.id)}">Editar</button>
                <button class="btn sm danger" data-action="device-del" data-id="${esc(d.id)}">Excluir</button>
            </td></tr>`;
    }).join('');
}

function renderTags() {
    const q = $('tagSearch').value.trim().toLowerCase();
    const dev = $('tagDeviceFilter').value;
    const tags = state.tags.filter((t) =>
        (!dev || t.device_id === dev) &&
        (!q || t.name.toLowerCase().includes(q) || t.id.toLowerCase().includes(q)));
    const body = $('tagsBody');
    if (!tags.length) {
        body.innerHTML = `<tr><td colspan="7" class="empty">${state.tags.length ? 'Nenhuma tag encontrada' : 'Nenhuma tag cadastrada'}</td></tr>`;
        return;
    }
    body.innerHTML = tags.map((t) => {
        const d = deviceById(t.device_id);
        const v = fmtValue(t);
        const addr = d && d.protocol === 'cip'
            ? esc(t.address)
            : `DB${t.db_number}.${t.data_type === 'BOOL' ? `DBX${t.offset}.${t.bit}` : `DBW${t.offset}`}`;
        return `<tr>
            <td><span class="badge ${v.bad ? 'err' : v.cls === 'true' ? 'on' : ''}">${esc(v.text)}</span></td>
            <td><code>${esc(t.id)}</code></td><td>${esc(t.name)}</td>
            <td>${d ? esc(d.name) : `<span class="sub">${esc(t.device_id)} (removido)</span>`}</td>
            <td>${esc(t.data_type)}</td><td>${addr}</td>
            <td class="right">
                <button class="btn sm" data-action="tag-edit" data-id="${esc(t.id)}">Editar</button>
                <button class="btn sm danger" data-action="tag-del" data-id="${esc(t.id)}">Excluir</button>
            </td></tr>`;
    }).join('');
}

function fmtBytes(n) {
    if (n == null) return '—';
    const u = ['B', 'KB', 'MB', 'GB', 'TB'];
    let i = 0;
    while (n >= 1024 && i < u.length - 1) { n /= 1024; i++; }
    return `${n.toFixed(i ? 1 : 0)} ${u[i]}`;
}

function renderNetwork() {
    const info = state.net;
    if (!info) { $('netGrid').innerHTML = '<p class="sub">A carregar…</p>'; return; }
    $('netBanner').innerHTML = info.writable ? '' :
        `<div class="banner">Sistema "${esc(info.os)}": a edição de rede só está disponível em Linux (requer permissões de root). Exibindo somente leitura.</div>`;
    const list = info.interfaces.filter((i) => state.showVirtual || i.kind === 'ethernet');
    if (!list.length) {
        $('netGrid').innerHTML = '<div class="panel"><p class="empty">Nenhuma porta ethernet encontrada.</p></div>';
        return;
    }
    $('netGrid').innerHTML = list.map((i) => {
        const link = i.carrier === true ? 'Link detectado' : i.carrier === false ? 'Sem cabo' : '—';
        const cls = i.up && i.carrier !== false ? 'online' : i.up ? 'connecting' : 'offline';
        const ips = i.addresses.map((a) =>
            `<li><code>${esc(a)}</code>${info.writable && i.kind !== 'loopback'
                ? `<div class="btn-group">
                    <button class="btn sm" data-action="net-edit-ip" data-iface="${esc(i.name)}" data-addr="${esc(a)}" data-gw="${esc(i.gateway || '')}">✏️ Editar</button>
                    <button class="btn sm danger" data-action="net-del-ip" data-iface="${esc(i.name)}" data-addr="${esc(a)}">🗑️</button>
                   </div>` : ''}</li>`).join('')
            || '<li class="sub">Sem endereços</li>';
        const dis = info.writable && i.kind !== 'loopback' ? '' : 'disabled';
        return `<div class="net-card">
            <header><h3>${esc(i.name)} <span class="sub">${esc(i.kind)}</span></h3>
                <span class="state"><span class="dot dot-${cls}"></span><span class="badge ${cls}">${i.up ? 'Ligada' : 'Desligada'}</span></span></header>
            <dl class="kv">
                <dt>Link</dt><dd>${link}${i.speed_mbps ? ` · ${i.speed_mbps} Mb/s` : ''}${i.duplex ? ` · ${esc(i.duplex)}` : ''}</dd>
                <dt>MAC</dt><dd>${esc(i.mac || '—')}</dd>
                <dt>MTU</dt><dd>${i.mtu}</dd>
                <dt>Gateway</dt><dd>${esc(i.gateway || '—')}</dd>
                <dt>RX / TX</dt><dd>${fmtBytes(i.rx_bytes)} / ${fmtBytes(i.tx_bytes)}</dd>
            </dl>
            <div class="sub">Endereços IP</div><ul class="ips">${ips}</ul>
            <div class="actions">
                <button class="btn sm" ${dis} data-action="net-toggle" data-iface="${esc(i.name)}" data-up="${i.up ? '0' : '1'}">${i.up ? 'Desligar' : 'Ligar'}</button>
                <button class="btn sm primary" ${dis} data-action="net-static" data-iface="${esc(i.name)}">Definir IP estático</button>
                <button class="btn sm" ${dis} data-action="net-add-ip" data-iface="${esc(i.name)}">+ IP adicional</button>
            </div></div>`;
    }).join('');
}

/* ---------- navegação ---------- */
let netTimer = null;
function navigate() {
    const page = (location.hash || '#dashboard').slice(1);
    state.page = ['dashboard', 'devices', 'tags', 'network'].includes(page) ? page : 'dashboard';
    document.querySelectorAll('.page').forEach((p) => p.classList.toggle('active', p.id === `page-${state.page}`));
    document.querySelectorAll('#nav a').forEach((a) => a.classList.toggle('active', a.dataset.page === state.page));
    clearInterval(netTimer);
    if (state.page === 'network') {
        loadNetwork();
        netTimer = setInterval(loadNetwork, 3000);
    }
    console.log(`${LOG} página: ${state.page}`);
    render();
}

/* ---------- WebSocket (tempo real) ---------- */
function connectWS() {
    const url = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`;
    console.log(`${LOG} conectando WebSocket ${url}`);
    const ws = new WebSocket(url);
    const setWs = (on) => {
        $('ws-dot').className = `dot ${on ? 'dot-online' : 'dot-offline'}`;
        $('ws-text').textContent = `Tempo real: ${on ? 'conectado' : 'desconectado'}`;
    };
    ws.onopen = () => { console.log(`${LOG} WebSocket conectado`); setWs(true); };
    ws.onclose = () => { console.warn(`${LOG} WebSocket fechado; nova tentativa em 3 s`); setWs(false); setTimeout(connectWS, 3000); };
    ws.onerror = (e) => console.error(`${LOG} erro no WebSocket`, e);
    ws.onmessage = (ev) => {
        let tags;
        try { tags = JSON.parse(ev.data); } catch (e) { console.error(`${LOG} mensagem inválida`, ev.data); return; }
        if (!Array.isArray(tags)) return;
        for (const t of tags) state.live[t.id] = { value: t.value, quality: t.quality, timestamp: t.timestamp };
        if (state.page === 'dashboard' || state.page === 'tags') render();
    };
}

/* ---------- diálogos: dispositivo ---------- */
function syncDeviceForm() {
    const s7 = $('devProtocol').value === 's7';
    document.querySelectorAll('#dlgDevice .s7-only').forEach((e) => e.classList.toggle('hidden', !s7));
}

function openDeviceDialog(id) {
    const d = id ? deviceById(id) : null;
    $('formDevice').reset();
    $('devError').textContent = '';
    $('devTitle').textContent = d ? `Editar dispositivo: ${d.id}` : 'Novo dispositivo';
    $('devId').disabled = !!d;
    if (d) {
        $('devId').value = d.id; $('devName').value = d.name; $('devProtocol').value = d.protocol;
        $('devIp').value = d.ip_address; $('devRack').value = d.rack; $('devSlot').value = d.slot;
    }
    $('formDevice').dataset.mode = d ? 'edit' : 'create';
    syncDeviceForm();
    $('dlgDevice').showModal();
}

async function submitDevice(e) {
    e.preventDefault();
    const edit = $('formDevice').dataset.mode === 'edit';
    const dev = {
        id: $('devId').value.trim(),
        name: $('devName').value.trim(),
        protocol: $('devProtocol').value,
        ip_address: $('devIp').value.trim(),
        rack: parseInt($('devRack').value, 10) || 0,
        slot: parseInt($('devSlot').value, 10) || 0,
    };
    try {
        await api(edit ? 'PUT' : 'POST', edit ? `/api/devices/${encodeURIComponent(dev.id)}` : '/api/devices', dev);
        $('dlgDevice').close();
        toast(edit ? 'Dispositivo atualizado' : 'Dispositivo criado', 'success');
        await loadConfig();
        loadStatuses();
    } catch (err) { $('devError').textContent = err.message; }
}

async function deleteDevice(id) {
    const n = state.tags.filter((t) => t.device_id === id).length;
    if (!(await confirmDialog('Excluir dispositivo', `Excluir "${id}"? ${n} tag(s) associada(s) também serão excluídas.`, 'Excluir'))) return;
    try {
        await api('DELETE', `/api/devices/${encodeURIComponent(id)}`);
        toast('Dispositivo excluído', 'success');
        await loadConfig();
    } catch (err) { toast(err.message, 'error'); }
}

/* ---------- diálogos: tag ---------- */
function syncTagForm() {
    const d = deviceById($('tagDevice').value);
    const cip = d && d.protocol === 'cip';
    const bool = $('tagType').value === 'BOOL';
    document.querySelectorAll('#dlgTag .s7-only').forEach((e) => e.classList.toggle('hidden', cip));
    document.querySelectorAll('#dlgTag .cip-only').forEach((e) => e.classList.toggle('hidden', !cip));
    document.querySelectorAll('#dlgTag .bool-only').forEach((e) => e.classList.toggle('hidden', cip || !bool));
}

function openTagDialog(id) {
    if (!state.devices.length) { toast('Cadastre um dispositivo antes de criar tags', 'error'); return; }
    const t = id ? state.tags.find((x) => x.id === id) : null;
    $('formTag').reset();
    $('tagError').textContent = '';
    $('tagDevice').innerHTML = state.devices.map((d) => `<option value="${esc(d.id)}">${esc(d.name)} (${esc(d.protocol.toUpperCase())} · ${esc(d.ip_address)})</option>`).join('');
    $('tagTitle').textContent = t ? `Editar tag: ${t.id}` : 'Nova tag';
    $('tagId').disabled = !!t;
    if (t) {
        $('tagId').value = t.id; $('tagName').value = t.name; $('tagDevice').value = t.device_id;
        $('tagType').value = t.data_type; $('tagBit').value = t.bit; $('tagDb').value = t.db_number;
        $('tagOffset').value = t.offset; $('tagAddress').value = t.address;
    } else {
        const f = $('tagDeviceFilter').value;
        if (f) $('tagDevice').value = f;
    }
    $('formTag').dataset.mode = t ? 'edit' : 'create';
    syncTagForm();
    $('dlgTag').showModal();
}

async function submitTag(e) {
    e.preventDefault();
    const edit = $('formTag').dataset.mode === 'edit';
    const tag = {
        id: $('tagId').value.trim(),
        name: $('tagName').value.trim(),
        device_id: $('tagDevice').value,
        data_type: $('tagType').value,
        bit: parseInt($('tagBit').value, 10) || 0,
        db_number: parseInt($('tagDb').value, 10) || 0,
        offset: parseInt($('tagOffset').value, 10) || 0,
        address: $('tagAddress').value.trim(),
    };
    const d = deviceById(tag.device_id);
    if (d && d.protocol === 'cip') { tag.db_number = 0; tag.offset = 0; tag.bit = 0; }
    else { tag.address = ''; if (tag.data_type !== 'BOOL') tag.bit = 0; }
    try {
        await api(edit ? 'PUT' : 'POST', edit ? `/api/tags/${encodeURIComponent(tag.id)}` : '/api/tags', tag);
        $('dlgTag').close();
        toast(edit ? 'Tag atualizada' : 'Tag criada', 'success');
        await loadConfig();
    } catch (err) { $('tagError').textContent = err.message; }
}

async function deleteTag(id) {
    if (!(await confirmDialog('Excluir tag', `Excluir a tag "${id}"?`, 'Excluir'))) return;
    try {
        await api('DELETE', `/api/tags/${encodeURIComponent(id)}`);
        delete state.live[id];
        toast('Tag excluída', 'success');
        await loadConfig();
    } catch (err) { toast(err.message, 'error'); }
}

/* ---------- diálogos: rede ---------- */
function openNetDialog(iface, mode, currentAddr = '', currentGw = '') {
    $('formNet').reset();
    $('netError').textContent = '';
    $('netIface').value = iface;
    $('netMode').value = mode;
    $('netOldAddr').value = currentAddr;
    
    if (mode === 'static') {
        $('netTitle').textContent = `Definir IP estático – ${iface}`;
        $('netModeHint').textContent = 'Substitui TODOS os endereços IPv4 desta interface. Se você acessa este painel por ela, a conexão será perdida e deverá reabrir no novo IP.';
    } else if (mode === 'edit') {
        $('netTitle').textContent = `Editar IP (${currentAddr}) – ${iface}`;
        $('netModeHint').textContent = 'Substitui em tempo real este endereço IP específico pelo novo informado abaixo.';
        $('netAddr').value = currentAddr;
        $('netGw').value = currentGw;
    } else {
        $('netTitle').textContent = `IP adicional – ${iface}`;
        $('netModeHint').textContent = 'Adiciona um endereço IPv4 secundário, mantendo os atuais.';
    }
    
    $('dlgNet').showModal();
}

async function submitNet(e) {
    e.preventDefault();
    const iface = $('netIface').value;
    const mode = $('netMode').value;
    const oldAddr = $('netOldAddr').value;
    const newAddr = $('netAddr').value.trim();
    const gateway = $('netGw').value.trim();
    const base = `/api/network/interfaces/${encodeURIComponent(iface)}`;

    try {
        if (mode === 'static') {
            await api('PUT', `${base}/ipv4`, { address: newAddr, gateway });
        } else if (mode === 'edit') {
            await api('PUT', `${base}/addresses`, { old_address: oldAddr, new_address: newAddr, gateway });
        } else {
            await api('POST', `${base}/addresses`, { address: newAddr, gateway });
        }
        $('dlgNet').close();
        toast('Configuração aplicada em tempo real', 'success');
        loadNetwork();
    } catch (err) { $('netError').textContent = err.message; }
}

/* ---------- eventos ---------- */
document.addEventListener('click', async (ev) => {
    const el = ev.target.closest('[data-action]');
    if (!el) return;
    const { action, id, iface, addr, up } = el.dataset;
    switch (action) {
        case 'dlg-close': el.closest('dialog').close(); break;
        case 'device-new': openDeviceDialog(); break;
        case 'device-edit': openDeviceDialog(id); break;
        case 'device-del': deleteDevice(id); break;
        case 'tag-new': openTagDialog(); break;
        case 'tag-edit': openTagDialog(id); break;
        case 'tag-del': deleteTag(id); break;
        case 'net-refresh': loadNetwork(); break;
        case 'net-static': openNetDialog(iface, 'static'); break;
        case 'net-add-ip': openNetDialog(iface, 'add'); break;
        case 'net-edit-ip': openNetDialog(iface, 'edit', addr, el.dataset.gw || ''); break;
        case 'net-toggle':
            if (up === '0' && !(await confirmDialog('Desligar interface', `Desligar "${iface}"? Se for a porta usada para acessar este painel, o acesso será perdido.`, 'Desligar'))) break;
            try {
                await api('POST', `/api/network/interfaces/${encodeURIComponent(iface)}/state`, { up: up === '1' });
                toast(`Interface ${iface} ${up === '1' ? 'ligada' : 'desligada'}`, 'success');
            } catch (err) { toast(err.message, 'error'); }
            loadNetwork();
            break;
        case 'net-del-ip':
            if (!(await confirmDialog('Remover endereço', `Remover ${addr} de ${iface}?`, 'Remover'))) break;
            try {
                await api('DELETE', `/api/network/interfaces/${encodeURIComponent(iface)}/addresses?address=${encodeURIComponent(addr)}`);
                toast('Endereço removido', 'success');
            } catch (err) { toast(err.message, 'error'); }
            loadNetwork();
            break;
    }
});

window.addEventListener('hashchange', navigate);
$('formDevice').addEventListener('submit', submitDevice);
$('formTag').addEventListener('submit', submitTag);
$('formNet').addEventListener('submit', submitNet);
$('devProtocol').addEventListener('change', syncDeviceForm);
$('tagDevice').addEventListener('change', syncTagForm);
$('tagType').addEventListener('change', syncTagForm);
$('dashDeviceFilter').addEventListener('change', render);
$('tagDeviceFilter').addEventListener('change', render);
$('tagSearch').addEventListener('input', render);
$('netShowVirtual').addEventListener('change', (e) => { state.showVirtual = e.target.checked; render(); });

/* ---------- init ---------- */
console.log(`${LOG} iniciando`);
navigate();
loadConfig().catch((e) => toast(`Erro ao carregar configuração: ${e.message}`, 'error'));
loadStatuses();
setInterval(loadStatuses, 2000);
connectWS();
