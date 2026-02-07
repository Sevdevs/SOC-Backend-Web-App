const apiBase = "/api/incidents";

function $(id) {
  return document.getElementById(id);
}

function formatTime(value) {
  const date = new Date(value);
  return date.toLocaleString([], {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatChip(value) {
  const span = document.createElement("span");
  span.className = "chip";
  span.textContent = value;
  return span;
}

function formatSeverity(severity) {
  const span = document.createElement("span");
  span.className = `badge severity-${severity.toLowerCase()}`;
  span.textContent = severity;
  return span;
}

async function fetchJSON(url, options) {
  const response = await fetch(url, options);
  if (!response.ok) {
    throw new Error("Request failed");
  }
  return response.json();
}

function renderIncidents(items) {
  const table = $("incident-table");
  const empty = $("incident-empty");
  table.innerHTML = "";

  if (!items.length) {
    empty.style.display = "block";
    $("incident-count").textContent = "0 incidents";
    return;
  }

  empty.style.display = "none";
  $("incident-count").textContent = `${items.length} incidents`;

  items.forEach((incident) => {
    const row = document.createElement("div");
    row.className = "table-row";

    const idCell = document.createElement("a");
    idCell.href = `detail.html?id=${encodeURIComponent(incident.id)}`;
    idCell.className = "mono link";
    idCell.textContent = incident.id;

    const titleCell = document.createElement("span");
    titleCell.textContent = incident.title;

    const severityCell = document.createElement("span");
    severityCell.appendChild(formatSeverity(incident.severity));

    const statusCell = document.createElement("span");
    statusCell.className = "status";
    statusCell.textContent = incident.status;

    const ownerCell = document.createElement("span");
    ownerCell.textContent = incident.owner;

    const updatedCell = document.createElement("span");
    updatedCell.textContent = formatTime(incident.updatedAt);

    const tagsCell = document.createElement("span");
    tagsCell.className = "chip-row";
    incident.tags.slice(0, 3).forEach((tag) => tagsCell.appendChild(formatChip(tag)));

    row.append(idCell, titleCell, severityCell, statusCell, ownerCell, updatedCell, tagsCell);
    table.appendChild(row);
  });
}

async function loadList() {
  try {
    const params = new URLSearchParams();
    const search = $("filter-search");
    const severity = $("filter-severity");
    const status = $("filter-status");

    if (search && search.value.trim()) {
      params.set("q", search.value.trim());
    }
    if (severity && severity.value) {
      params.set("severity", severity.value);
    }
    if (status && status.value) {
      params.set("status", status.value);
    }

    const url = params.toString() ? `${apiBase}?${params}` : apiBase;
    const data = await fetchJSON(url);
    renderIncidents(data.items || []);
  } catch (error) {
    renderIncidents([]);
    const message = $("form-error");
    if (message) {
      message.textContent = "Unable to reach the API. Restart the Go server.";
    }
  }
}

function parseCommaField(value) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

function bindCreateForm() {
  const form = $("create-form");
  if (!form) {
    return;
  }

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const message = $("form-error");
    if (message) {
      message.textContent = "";
    }
    const formData = new FormData(form);
    const payload = {
      title: formData.get("title"),
      severity: formData.get("severity"),
      status: formData.get("status"),
      owner: formData.get("owner"),
      tags: parseCommaField(formData.get("tags") || ""),
      iocs: parseCommaField(formData.get("iocs") || ""),
    };

    try {
      await fetchJSON(apiBase, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      form.reset();
      await loadList();
    } catch (error) {
      if (message) {
        message.textContent = "Create failed. Ensure the API is running.";
      }
    }
  });
}

function bindFilters() {
  const search = $("filter-search");
  const severity = $("filter-severity");
  const status = $("filter-status");
  const clear = $("filter-clear");

  if (!search || !severity || !status || !clear) {
    return;
  }

  const trigger = () => loadList();

  search.addEventListener("input", () => {
    if (search.value.length === 0 || search.value.length > 2) {
      trigger();
    }
  });
  severity.addEventListener("change", trigger);
  status.addEventListener("change", trigger);
  clear.addEventListener("click", () => {
    search.value = "";
    severity.value = "";
    status.value = "";
    trigger();
  });
}

function renderDetail(incident) {
  $("detail-title").textContent = incident.title;
  $("detail-subtitle").textContent = `Opened ${formatTime(incident.createdAt)}`;
  $("detail-id").textContent = incident.id;
  $("detail-updated").textContent = `Updated ${formatTime(incident.updatedAt)}`;

  const severitySelect = $("detail-severity");
  severitySelect.value = incident.severity;
  const statusSelect = $("detail-status");
  statusSelect.value = incident.status;
  const ownerInput = $("detail-owner");
  ownerInput.value = incident.owner;

  const iocs = $("detail-iocs");
  iocs.innerHTML = "";
  (incident.iocs || []).forEach((item) => iocs.appendChild(formatChip(item)));
  if (!incident.iocs || incident.iocs.length === 0) {
    iocs.appendChild(formatChip("No indicators"));
  }

  const tags = $("detail-tags");
  tags.innerHTML = "";
  (incident.tags || []).forEach((item) => tags.appendChild(formatChip(item)));
  if (!incident.tags || incident.tags.length === 0) {
    tags.appendChild(formatChip("No tags"));
  }

  const notes = $("note-list");
  const emptyNotes = $("note-empty");
  notes.innerHTML = "";
  if (!incident.notes || incident.notes.length === 0) {
    emptyNotes.style.display = "block";
  } else {
    emptyNotes.style.display = "none";
    incident.notes.forEach((note) => {
      const card = document.createElement("div");
      card.className = "note";
      const meta = document.createElement("div");
      meta.className = "note-meta";
      meta.textContent = `${note.author} · ${formatTime(note.createdAt)}`;
      const body = document.createElement("p");
      body.textContent = note.body;
      card.append(meta, body);
      notes.appendChild(card);
    });
  }
}

async function updateIncident(id, payload) {
  return fetchJSON(`${apiBase}/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

function bindDetailControls(id) {
  const severitySelect = $("detail-severity");
  const statusSelect = $("detail-status");
  const ownerInput = $("detail-owner");

  const update = async () => {
    const updated = await updateIncident(id, {
      severity: severitySelect.value,
      status: statusSelect.value,
      owner: ownerInput.value,
    });
    renderDetail(updated);
  };

  severitySelect.addEventListener("change", update);
  statusSelect.addEventListener("change", update);
  ownerInput.addEventListener("blur", update);

  const form = $("note-form");
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const formData = new FormData(form);
    const payload = {
      author: formData.get("author"),
      body: formData.get("body"),
    };
    const updated = await fetchJSON(`${apiBase}/${id}/notes`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    form.reset();
    renderDetail(updated);
  });
}

async function loadDetail() {
  const params = new URLSearchParams(window.location.search);
  const id = params.get("id");
  if (!id) {
    $("detail-title").textContent = "Incident not found";
    return;
  }

  try {
    const incident = await fetchJSON(`${apiBase}/${id}`);
    renderDetail(incident);
    bindDetailControls(id);
  } catch (error) {
    $("detail-title").textContent = "Unable to reach the API";
    $("detail-subtitle").textContent = "Restart the Go server and try again.";
  }
}

const page = document.body.dataset.page;
if (page === "list") {
  loadList();
  bindCreateForm();
  bindFilters();
}

if (page === "detail") {
  loadDetail();
}
