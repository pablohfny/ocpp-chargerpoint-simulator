// Parceiros tab: the OCPI partner console — profile CRUD, manual commands and
// the per-partner event feed with the cdr_token echo indicator.
Sim.tabs.parceiros = (function () {
    const FORM_FIELDS = ['slug', 'name', 'partyId', 'countryCode', 'tokenToCallUs', 'tokenExpected', 'ocpiBaseUrl', 'publicBaseUrl'];

    let partners = [];
    let defaults = {};
    let selectedSlug = null;
    let editingSlug = null;
    let lastEventId = 0;
    let renderedEvents = 0;

    function init() {
        document.getElementById('newPartnerButton').addEventListener('click', () => {
            selectedSlug = null;
            renderPartnerList();
            renderCommands();
            fillForm(null);
        });

        document.getElementById('cancelPartnerButton').addEventListener('click', () => {
            fillForm(findPartner(selectedSlug));
        });

        document.getElementById('partnerForm').addEventListener('submit', savePartner);
        document.getElementById('deletePartnerButton').addEventListener('click', deletePartner);
        document.getElementById('clearEventsButton').addEventListener('click', clearEvents);
    }

    async function activate() {
        await loadDefaults();
        await loadPartners();

        if (!selectedSlug && partners.length > 0) {
            selectPartner(partners[0].slug);
        } else {
            fillForm(findPartner(selectedSlug));
        }
    }

    function refresh() {
        pollEvents();
    }

    async function loadDefaults() {
        try {
            defaults = await Sim.request(Sim.OCPI_API + '/defaults');
        } catch (error) {
            defaults = {};
        }
    }

    async function loadPartners() {
        try {
            partners = await Sim.request(Sim.OCPI_API + '/partners');
        } catch (error) {
            partners = [];
            Sim.setMessage('partnerFormMessage', error.message, 'error');
        }
        renderPartnerList();
    }

    function findPartner(slug) {
        return partners.find((partner) => partner.slug === slug) || null;
    }

    function renderPartnerList() {
        const list = document.getElementById('partnerList');

        if (partners.length === 0) {
            Sim.emptyState(list, 'Nenhum parceiro cadastrado.');
            return;
        }

        Sim.clear(list);
        partners.forEach((partner) => {
            const card = Sim.element('div', 'partner-card' + (partner.slug === selectedSlug ? ' selected' : ''));
            card.appendChild(Sim.element('div', 'partner-name', partner.name));
            card.appendChild(Sim.element('div', 'partner-meta',
                partner.countryCode + '/' + partner.partyId + ' · ' + partner.slug));
            card.addEventListener('click', () => selectPartner(partner.slug));
            list.appendChild(card);
        });
    }

    function selectPartner(slug) {
        selectedSlug = slug;
        lastEventId = 0;
        renderedEvents = 0;

        Sim.emptyState(document.getElementById('partnerEvents'), 'Carregando eventos...');
        document.getElementById('clearEventsButton').disabled = false;

        renderPartnerList();
        fillForm(findPartner(slug));
        renderCommands();
        pollEvents();
    }

    function fieldElement(name) {
        return document.getElementById('field' + name.charAt(0).toUpperCase() + name.slice(1));
    }

    function fillForm(partner) {
        editingSlug = partner ? partner.slug : null;

        FORM_FIELDS.forEach((name) => {
            fieldElement(name).value = partner ? (partner[name] || '') : '';
        });

        fieldElement('slug').disabled = Boolean(partner);
        document.getElementById('partnerFormTitle').textContent = partner ? 'Editar parceiro' : 'Novo parceiro';
        document.getElementById('deletePartnerButton').style.display = partner ? 'inline-block' : 'none';

        if (!partner) {
            fieldElement('ocpiBaseUrl').value = defaults.ocpiBaseUrl || '';
            fieldElement('publicBaseUrl').value = defaults.publicBaseUrl || '';
        }
        Sim.setMessage('partnerFormMessage', '');
    }

    async function savePartner(event) {
        event.preventDefault();

        const body = {};
        FORM_FIELDS.forEach((name) => {
            body[name] = fieldElement(name).value.trim();
        });

        const isEditing = Boolean(editingSlug);
        const path = isEditing ? '/partners/' + editingSlug : '/partners';

        document.getElementById('savePartnerButton').disabled = true;
        try {
            const saved = await Sim.request(Sim.OCPI_API + path, {
                method: isEditing ? 'PUT' : 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body)
            });
            Sim.setMessage('partnerFormMessage', 'Parceiro salvo.', 'success');
            await loadPartners();
            selectPartner(saved.slug);
        } catch (error) {
            Sim.setMessage('partnerFormMessage', error.message, 'error');
        } finally {
            document.getElementById('savePartnerButton').disabled = false;
        }
    }

    async function deletePartner() {
        if (!editingSlug || !confirm('Excluir o parceiro ' + editingSlug + '?')) return;

        try {
            await Sim.request(Sim.OCPI_API + '/partners/' + editingSlug, { method: 'DELETE' });
            selectedSlug = null;
            await loadPartners();
            fillForm(null);
            renderCommands();
            Sim.emptyState(document.getElementById('partnerEvents'), 'Nenhum evento ainda.');
            document.getElementById('clearEventsButton').disabled = true;
        } catch (error) {
            Sim.setMessage('partnerFormMessage', error.message, 'error');
        }
    }

    // renderCommands builds the manual START/STOP form for the selected partner.
    function renderCommands() {
        const partner = findPartner(selectedSlug);
        const body = document.getElementById('partnerCommandBody');
        const title = document.getElementById('partnerCommandTitle');

        if (!partner) {
            title.textContent = 'Comandos';
            Sim.emptyState(body, 'Selecione um parceiro para enviar comandos.');
            return;
        }

        title.textContent = 'Comandos · ' + partner.name;
        Sim.clear(body);

        const grid = Sim.element('div', 'form-grid');
        grid.appendChild(commandField('commandLocation', 'Location ID', defaults.locationId || ''));
        grid.appendChild(commandField('commandEvse', 'EVSE UID', defaults.evseUid || ''));
        grid.appendChild(commandField('commandConnector', 'Connector ID', ''));
        grid.appendChild(commandField('commandSession', 'Session ID (parar)', ''));
        body.appendChild(grid);

        const row = Sim.element('div', 'button-row');
        const startButton = Sim.element('button', 'btn btn-success', 'Iniciar recarga');
        startButton.type = 'button';
        startButton.addEventListener('click', startSession);
        const stopButton = Sim.element('button', 'btn btn-danger', 'Parar recarga');
        stopButton.type = 'button';
        stopButton.addEventListener('click', stopSession);
        row.appendChild(startButton);
        row.appendChild(stopButton);
        body.appendChild(row);

        body.appendChild(Sim.element('div', 'hint',
            'Receiver: ' + partner.publicBaseUrl + '/ocpi/p/' + partner.slug + '/receiver/2.2.1/...'));
        body.appendChild(Sim.element('div', 'message', ''));
        body.lastChild.id = 'partnerCommandMessage';
    }

    function commandField(id, label, value) {
        const field = Sim.element('div', 'field');
        const labelNode = Sim.element('label', null, label);
        labelNode.htmlFor = id;
        const input = document.createElement('input');
        input.id = id;
        input.value = value;
        field.appendChild(labelNode);
        field.appendChild(input);
        return field;
    }

    async function startSession() {
        Sim.setMessage('partnerCommandMessage', 'Enviando START_SESSION...');

        try {
            const result = await Sim.post(Sim.OCPI_API + '/partners/' + selectedSlug + '/commands/start', {
                locationId: document.getElementById('commandLocation').value.trim(),
                evseUid: document.getElementById('commandEvse').value.trim(),
                connectorId: document.getElementById('commandConnector').value.trim()
            });
            Sim.setMessage('partnerCommandMessage', describeResult(result), result.statusCode >= 400 ? 'error' : 'success');
        } catch (error) {
            Sim.setMessage('partnerCommandMessage', error.message, 'error');
        } finally {
            pollEvents();
        }
    }

    async function stopSession() {
        const sessionId = document.getElementById('commandSession').value.trim();
        if (!sessionId) {
            Sim.setMessage('partnerCommandMessage', 'Informe o Session ID para parar a recarga.', 'error');
            return;
        }

        Sim.setMessage('partnerCommandMessage', 'Enviando STOP_SESSION...');
        try {
            const result = await Sim.post(Sim.OCPI_API + '/partners/' + selectedSlug + '/commands/stop', {
                sessionId: sessionId
            });
            Sim.setMessage('partnerCommandMessage', describeResult(result), result.statusCode >= 400 ? 'error' : 'success');
        } catch (error) {
            Sim.setMessage('partnerCommandMessage', error.message, 'error');
        } finally {
            pollEvents();
        }
    }

    function describeResult(result) {
        if (result.event && result.event.error) {
            return 'Falha ao contatar a plataforma: ' + result.event.error;
        }
        const parts = ['HTTP ' + (result.statusCode || '?')];
        if (result.contractId) parts.push('contract_id ' + result.contractId);
        if (result.commandUid) parts.push('uid ' + result.commandUid);
        return parts.join(' · ');
    }

    async function clearEvents() {
        if (!selectedSlug) return;

        try {
            await Sim.request(Sim.OCPI_API + '/partners/' + selectedSlug + '/events', { method: 'DELETE' });
            lastEventId = 0;
            renderedEvents = 0;
            Sim.emptyState(document.getElementById('partnerEvents'), 'Nenhum evento ainda.');
        } catch (error) {
            Sim.setMessage('partnerFormMessage', error.message, 'error');
        }
    }

    async function pollEvents() {
        if (!selectedSlug) return;

        const slugAtRequest = selectedSlug;
        let payload;
        try {
            payload = await Sim.request(Sim.OCPI_API + '/partners/' + slugAtRequest + '/events?after=' + lastEventId);
        } catch (error) {
            return;
        }

        // The selection may have changed while the request was in flight.
        if (slugAtRequest !== selectedSlug) return;

        const list = document.getElementById('partnerEvents');
        if (renderedEvents === 0 && payload.events.length === 0) {
            Sim.emptyState(list, 'Nenhum evento ainda.');
            return;
        }

        if (renderedEvents === 0) Sim.clear(list);

        payload.events.forEach((event) => {
            list.insertBefore(Sim.renderEvent(event), list.firstChild);
            renderedEvents++;
        });
        lastEventId = payload.lastId || lastEventId;
    }

    return { init: init, activate: activate, refresh: refresh };
})();
