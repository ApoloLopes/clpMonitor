// Obtém dinamicamente o IP/Porta atual (permite acessar de outras máquinas na rede)
const wsUrl = `ws://${window.location.host}/ws`;
let socket = new WebSocket(wsUrl);

const statusDot = document.getElementById('ws-status');
const tagsContainer = document.getElementById('tags-container');

// --- Lógica de Preenchimento Dinâmico ---

const tagDeviceSelect = document.getElementById('tagDevice');

// Função para buscar dispositivos da API e preencher o select
function loadDevicesForDropdown() {
    fetch('/api/config')
        .then(response => response.json())
        .then(data => {
            tagDeviceSelect.innerHTML = ''; // Limpa opções antigas

            if (data.devices && data.devices.length > 0) {
                data.devices.forEach(dev => {
                    const option = document.createElement('option');
                    option.value = dev.id;
                    // Mostra o Nome, Protocolo e IP para facilitar a identificação
                    option.textContent = `${dev.name} (${dev.protocol.toUpperCase()} - ${dev.ip_address})`;
                    tagDeviceSelect.appendChild(option);
                });
            } else {
                tagDeviceSelect.innerHTML = '<option value="" disabled selected>Nenhum dispositivo cadastrado</option>';
            }
        })
        .catch(error => console.error('Erro ao carregar dispositivos:', error));
}

// Quando a página terminar de carregar, executa a busca
document.addEventListener('DOMContentLoaded', () => {
    loadDevicesForDropdown();
});

// Para garantir que a lista esteja sempre atualizada, recarrega também sempre que o utilizador abrir a Modal de Tags
document.getElementById('addTagModal').addEventListener('show.bs.modal', loadDevicesForDropdown);


// Evento: Conexão estabelecida com o Fiber
socket.onopen = () => {
    console.log("WebSocket Conectado!");
    statusDot.classList.replace('status-offline', 'status-online');
};

// Evento: Conexão perdida
socket.onclose = () => {
    console.log("WebSocket Desconectado.");
    statusDot.classList.replace('status-online', 'status-offline');
    // Em um cenário de Pleno, aqui colocaríamos um setTimeout para tentar reconectar (Auto-Reconnect)
};

// Evento: Mensagem recebida do Go (o nosso Worker enviou o JSON)
socket.onmessage = (event) => {
    const tags = JSON.parse(event.data);

    tags.forEach(tag => {
        // Formata o timestamp vindo do Go (ISO 8601) para hora legível
        const timeString = new Date(tag.timestamp).toLocaleTimeString();
        let displayValue = tag.value;

        // Formatação específica para números flutuantes
        if (typeof tag.value === 'number' && !Number.isInteger(tag.value)) {
            displayValue = tag.value.toFixed(2);
        }

        // Verifica se o card desta tag já existe na tela
        const existingCard = document.getElementById(`card-${tag.id}`);

        if (existingCard) {
            // Atualiza os dados do card existente (alta performance)
            const valueElement = document.getElementById(`value-${tag.id}`);
            const timeElement = document.getElementById(`time-${tag.id}`);
            const cardBody = document.getElementById(`body-${tag.id}`);

            valueElement.innerText = tag.quality ? displayValue : "ERRO";
            timeElement.innerText = timeString;

            // Altera a cor se a qualidade da leitura cair
            if (tag.quality) {
                cardBody.classList.remove('bg-danger', 'text-white');
                cardBody.classList.add('bg-white', 'text-dark');
            } else {
                cardBody.classList.remove('bg-white', 'text-dark');
                cardBody.classList.add('bg-danger', 'text-white');
            }

        } else {
            // Cria o card pela primeira vez
            const cardHTML = `
                <div class="col-md-4 mb-3" id="card-${tag.id}">
                    <div class="card shadow-sm h-100">
                        <div class="card-header d-flex justify-content-between align-items-center">
                            <strong>${tag.name}</strong>
                            <span class="badge bg-secondary">${tag.data_type}</span>
                        </div>
                        <div class="card-body bg-white text-dark text-center" id="body-${tag.id}">
                            <div class="card-value" id="value-${tag.id}">${tag.quality ? displayValue : "ERRO"}</div>
                        </div>
                        <div class="card-footer text-muted text-end small">
                            Atualizado: <span id="time-${tag.id}">${timeString}</span>
                        </div>
                    </div>
                </div>
            `;
            // Adiciona o novo card ao container principal
            tagsContainer.insertAdjacentHTML('beforeend', cardHTML);
        }
    });
};

// --- Lógica de Configuração via API REST ---

const btnSaveTag = document.getElementById('btnSaveTag');

btnSaveTag.addEventListener('click', () => {
    // 1. Capturar os valores do formulário
    const tagData = {
        id: document.getElementById('tagId').value,
        device_id: document.getElementById('tagDevice').value,
        name: document.getElementById('tagName').value,
        address: document.getElementById('tagName').value,
        db_number: parseInt(document.getElementById('tagDb').value),
        offset: parseInt(document.getElementById('tagOffset').value),
        data_type: document.getElementById('tagDataType').value,
        bit: parseInt(document.getElementById('tagBit').value) || 0
    };

    // 2. Enviar via POST para a API do Go
    fetch('/api/tags', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(tagData)
    })
    .then(response => {
        if (response.ok) {
            alert('Tag adicionada com sucesso no ficheiro JSON!');
            
            // Limpa o formulário e fecha a modal
            document.getElementById('formAddTag').reset();
            const modalElement = document.getElementById('addTagModal');
            const modalInstance = bootstrap.Modal.getInstance(modalElement);
            modalInstance.hide();
        } else {
            alert('Erro ao guardar a Tag. Verifique o console.');
            console.error('Resposta da API:', response);
        }
    })
    .catch(error => {
        console.error('Erro de rede ao tentar guardar:', error);
        alert('Erro de comunicação com o servidor.');
    });
});

// --- Lógica de Configuração de Dispositivos ---

const btnSaveDevice = document.getElementById('btnSaveDevice');

btnSaveDevice.addEventListener('click', () => {
    const deviceData = {
        id: document.getElementById('devId').value,
        name: document.getElementById('devName').value,
        protocol: document.getElementById('devProtocol').value,
        ip_address: document.getElementById('devIp').value,
        rack: parseInt(document.getElementById('devRack').value) || 0,
        slot: parseInt(document.getElementById('devSlot').value) || 0
    };

    fetch('/api/devices', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(deviceData)
    })
    .then(response => {
        if (response.ok) {
            // Limpa o formulário e fecha a modal
            document.getElementById('formAddDevice').reset();
            const modalElement = document.getElementById('addDeviceModal');
            const modalInstance = bootstrap.Modal.getInstance(modalElement);
            modalInstance.hide();
            
            // O Hot-Reload do backend já se vai encarregar de reiniciar os workers!
        } else {
            alert('Erro ao guardar o Dispositivo. Verifique o console.');
            console.error('Resposta da API:', response);
        }
    })
    .catch(error => {
        console.error('Erro de rede ao tentar guardar:', error);
        alert('Erro de comunicação com o servidor.');
    });
});