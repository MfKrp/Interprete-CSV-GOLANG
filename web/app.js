(() => {
  "use strict";

  const MAX_FILE_BYTES = 10 * 1024 * 1024;
  const $ = (selector) => document.querySelector(selector);
  const elements = {
    connectionDot: $("#connection-dot"),
    connectionStatus: $("#connection-status"),
    refreshConnection: $("#refresh-connection"),
    dropZone: $("#drop-zone"),
    fileInput: $("#file-input"),
    browseFile: $("#browse-file"),
    selectedFile: $("#selected-file"),
    selectedName: $("#selected-name"),
    selectedSize: $("#selected-size"),
    clearFile: $("#clear-file"),
    uploadButton: $("#upload-button"),
    uploadMessage: $("#upload-message"),
    datasetSection: $("#dataset-section"),
    emptyDataset: $("#empty-dataset"),
    datasetFileName: $("#dataset-file-name"),
    rowCount: $("#row-count"),
    columnCount: $("#column-count"),
    datasetSize: $("#dataset-size"),
    columnList: $("#column-list"),
    previewCount: $("#preview-count"),
    previewTable: $("#preview-table"),
    questionInput: $("#question-input"),
    questionLength: $("#question-length"),
    queryButton: $("#query-button"),
    queryError: $("#query-error"),
    resultPanel: $("#result-panel"),
    resultQuestion: $("#result-question"),
    resultOperation: $("#result-operation"),
    resultParameters: $("#result-parameters"),
    scalarResult: $("#scalar-result"),
    scalarValue: $("#scalar-value"),
    groupResult: $("#group-result"),
    groupTable: $("#group-table"),
    toast: $("#toast"),
    suggestions: [...document.querySelectorAll(".suggestion")],
  };

  const state = { file: null, datasetId: "", metadata: null, uploading: false, querying: false, toastTimer: 0 };

  function formatBytes(bytes) {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  }

  function formatNumber(value) {
    if (typeof value !== "number" || !Number.isFinite(value)) return String(value ?? "—");
    return new Intl.NumberFormat("es-AR", { maximumFractionDigits: 2 }).format(value);
  }

  function setMessage(element, message, kind = "") {
    element.textContent = message;
    element.classList.toggle("error", kind === "error");
    element.classList.toggle("success", kind === "success");
  }

  function showToast(message) {
    elements.toast.textContent = message;
    elements.toast.hidden = false;
    window.clearTimeout(state.toastTimer);
    state.toastTimer = window.setTimeout(() => { elements.toast.hidden = true; }, 3600);
  }

  async function requestJSON(url, options = {}) {
    let response;
    try {
      response = await fetch(url, options);
    } catch (error) {
      throw new Error("No se pudo conectar con el backend. Comprobá que CSV Copilot esté iniciado.");
    }
    let body;
    try {
      body = await response.json();
    } catch (error) {
      throw new Error("El backend devolvió una respuesta que no se pudo leer.");
    }
    if (!response.ok) {
      throw new Error(body.error || `La solicitud falló (${response.status}).`);
    }
    return body;
  }

  async function checkConnection() {
    elements.connectionStatus.textContent = "Comprobando conexión";
    elements.connectionDot.className = "connection-dot";
    try {
      const health = await requestJSON("/health", { method: "GET", cache: "no-store" });
      if (health.status !== "ok") throw new Error("Respuesta de salud inesperada");
      elements.connectionStatus.textContent = "Backend conectado";
      elements.connectionDot.className = "connection-dot online";
    } catch (error) {
      elements.connectionStatus.textContent = "Backend desconectado";
      elements.connectionDot.className = "connection-dot offline";
    }
  }

  function updateUploadButton() {
    elements.uploadButton.disabled = !state.file || state.uploading || state.querying;
  }

  function updateQueryButton() {
    const hasQuestion = elements.questionInput.value.trim().length > 0;
    elements.queryButton.disabled = !state.datasetId || !hasQuestion || state.querying || state.uploading;
  }

  function clearSelectedFile() {
    state.file = null;
    elements.fileInput.value = "";
    elements.selectedFile.hidden = true;
    elements.dropZone.hidden = false;
    setMessage(elements.uploadMessage, "");
    updateUploadButton();
  }

  function selectFile(file) {
    if (!file || state.uploading) return;
    setMessage(elements.uploadMessage, "");
    if (!file.name.toLowerCase().endsWith(".csv")) {
      state.file = null;
      elements.fileInput.value = "";
      elements.selectedFile.hidden = true;
      elements.dropZone.hidden = false;
      setMessage(elements.uploadMessage, "Seleccioná un archivo con extensión .csv.", "error");
      updateUploadButton();
      return;
    }
    if (file.size === 0) {
      state.file = null;
      elements.fileInput.value = "";
      elements.selectedFile.hidden = true;
      elements.dropZone.hidden = false;
      setMessage(elements.uploadMessage, "El archivo está vacío.", "error");
      updateUploadButton();
      return;
    }
    if (file.size > MAX_FILE_BYTES) {
      state.file = null;
      elements.fileInput.value = "";
      elements.selectedFile.hidden = true;
      elements.dropZone.hidden = false;
      setMessage(elements.uploadMessage, "El archivo supera el límite de 10 MB.", "error");
      updateUploadButton();
      return;
    }
    state.file = file;
    elements.selectedName.textContent = file.name;
    elements.selectedSize.textContent = formatBytes(file.size);
    elements.selectedFile.hidden = false;
    elements.dropZone.hidden = true;
    updateUploadButton();
  }

  function setUploading(isUploading) {
    state.uploading = isUploading;
    elements.fileInput.disabled = isUploading;
    elements.browseFile.disabled = isUploading;
    elements.clearFile.disabled = isUploading;
    elements.dropZone.setAttribute("aria-disabled", String(isUploading));
    elements.uploadButton.classList.toggle("is-loading", isUploading);
    elements.uploadButton.querySelector(".button-label").textContent = isUploading ? "Cargando…" : "Cargar archivo";
    updateUploadButton();
    updateQueryButton();
  }

  async function uploadFile() {
    if (!state.file || state.uploading || state.querying) return;
    const file = state.file;
    setUploading(true);
    setMessage(elements.uploadMessage, "Subiendo y procesando CSV…");
    elements.queryError.hidden = true;
    try {
      const form = new FormData();
      form.append("file", file, file.name);
      const uploaded = await requestJSON("/api/datasets", { method: "POST", body: form });
      if (!uploaded.id) throw new Error("El backend no devolvió el identificador del dataset.");
      const metadata = await requestJSON(`/api/datasets/${encodeURIComponent(uploaded.id)}`, { method: "GET" });
      state.datasetId = uploaded.id;
      state.metadata = metadata;
      renderDataset(metadata, file.size);
      clearResult();
      elements.queryError.hidden = true;
      elements.questionInput.disabled = false;
      elements.suggestions.forEach((button) => { button.disabled = false; });
      clearSelectedFile();
      setMessage(elements.uploadMessage, "Archivo cargado correctamente.", "success");
      showToast("Dataset cargado. Ya podés hacer preguntas.");
      updateQueryButton();
    } catch (error) {
      setMessage(elements.uploadMessage, error.message, "error");
    } finally {
      setUploading(false);
    }
  }

  function createCell(tag, value, className = "") {
    const cell = document.createElement(tag);
    cell.textContent = value == null ? "" : String(value);
    if (className) cell.className = className;
    return cell;
  }

  function renderDataset(metadata, fileSize) {
    const columns = Array.isArray(metadata.columns) ? metadata.columns : [];
    const preview = Array.isArray(metadata.preview) ? metadata.preview : [];
    elements.emptyDataset.hidden = true;
    elements.datasetSection.hidden = false;
    elements.datasetFileName.textContent = metadata.filename || "Archivo CSV";
    elements.rowCount.textContent = formatNumber(metadata.row_count || 0);
    elements.columnCount.textContent = formatNumber(columns.length);
    elements.datasetSize.textContent = formatBytes(fileSize || 0);
    elements.datasetSize.title = `${metadata.filename || "CSV"} · ${formatBytes(fileSize || 0)}`;
    elements.columnList.replaceChildren();
    for (const column of columns) {
      const chip = document.createElement("span");
      chip.className = "column-chip";
      chip.append(createCell("span", column.name, "column-chip-name"));
      chip.append(createCell("span", column.type, "column-chip-type"));
      elements.columnList.append(chip);
    }
    elements.previewCount.textContent = `${preview.length} ${preview.length === 1 ? "fila" : "filas"}`;
    const head = elements.previewTable.querySelector("thead");
    const body = elements.previewTable.querySelector("tbody");
    head.replaceChildren();
    body.replaceChildren();
    const headerRow = document.createElement("tr");
    for (const column of columns) headerRow.append(createCell("th", column.name));
    head.append(headerRow);
    for (const record of preview) {
      const row = document.createElement("tr");
      for (const column of columns) row.append(createCell("td", record?.[column.name] ?? ""));
      body.append(row);
    }
    if (columns.length === 0) {
      const row = document.createElement("tr");
      row.append(createCell("td", "El dataset no tiene columnas para mostrar."));
      body.append(row);
    }
  }

  function setQuerying(isQuerying) {
    state.querying = isQuerying;
    elements.queryButton.classList.toggle("is-loading", isQuerying);
    elements.queryButton.querySelector(".button-label").textContent = isQuerying ? "Analizando…" : "Consultar";
    updateQueryButton();
    updateUploadButton();
  }

  function clearResult() {
    elements.resultPanel.hidden = true;
    elements.scalarResult.hidden = true;
    elements.groupResult.hidden = true;
  }

  const operationNames = {
    count_rows: "Conteo de filas",
    count_distinct: "Valores distintos",
    count_filtered: "Conteo con filtro",
    sum: "Suma",
    average: "Promedio",
    min: "Mínimo",
    max: "Máximo",
    group_by: "Agrupación",
  };
  const parameterNames = {
    column: "Columna",
    group_column: "Agrupar por",
    aggregation: "Agregación",
    target_column: "Columna objetivo",
    operator: "Operador",
    value: "Valor",
  };

  function renderResult(response) {
    const instruction = response.instruction || {};
    const result = response.result || {};
    elements.resultQuestion.textContent = response.question || "";
    elements.resultOperation.textContent = operationNames[instruction.operation] || instruction.operation || "Operación";
    elements.resultParameters.replaceChildren();
    for (const [key, label] of Object.entries(parameterNames)) {
      const value = instruction[key];
      if (value == null || value === "") continue;
      const chip = document.createElement("div");
      chip.className = "parameter-chip";
      chip.append(createCell("dt", label));
      chip.append(createCell("dd", value));
      elements.resultParameters.append(chip);
    }

    const groups = Array.isArray(result.groups) ? result.groups : null;
    elements.scalarResult.hidden = groups !== null;
    elements.groupResult.hidden = groups === null;
    if (groups !== null) {
      const groupTable = elements.groupTable;
      const head = groupTable.querySelector("thead");
      const body = groupTable.querySelector("tbody");
      head.replaceChildren();
      body.replaceChildren();
      const isCount = instruction.aggregation === "count";
      const header = document.createElement("tr");
      header.append(createCell("th", "Grupo"));
      header.append(createCell("th", isCount ? "Registros" : "Suma"));
      head.append(header);
      for (const group of groups) {
        const row = document.createElement("tr");
        row.append(createCell("td", group.key));
        row.append(createCell("td", formatNumber(isCount ? group.count : group.sum)));
        body.append(row);
      }
      if (groups.length === 0) {
        const row = document.createElement("tr");
        row.append(createCell("td", "Sin grupos para mostrar."));
        row.append(createCell("td", "—"));
        body.append(row);
      }
    } else {
      elements.scalarValue.textContent = formatNumber(result.value);
    }
    elements.resultPanel.hidden = false;
    elements.resultPanel.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }

  async function submitQuery() {
    const question = elements.questionInput.value.trim();
    if (!state.datasetId || !question || state.querying || state.uploading) return;
    setQuerying(true);
    elements.queryError.hidden = true;
    clearResult();
    try {
      const result = await requestJSON(`/api/datasets/${encodeURIComponent(state.datasetId)}/query`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question }),
      });
      renderResult(result);
    } catch (error) {
      elements.queryError.textContent = error.message;
      elements.queryError.hidden = false;
    } finally {
      setQuerying(false);
    }
  }

  elements.refreshConnection.addEventListener("click", checkConnection);
  elements.browseFile.addEventListener("click", (event) => { event.stopPropagation(); elements.fileInput.click(); });
  elements.dropZone.addEventListener("click", (event) => {
    if (event.target.closest("button")) return;
    elements.fileInput.click();
  });
  elements.dropZone.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") { event.preventDefault(); elements.fileInput.click(); }
  });
  elements.fileInput.addEventListener("change", () => selectFile(elements.fileInput.files?.[0]));
  elements.clearFile.addEventListener("click", clearSelectedFile);
  elements.uploadButton.addEventListener("click", uploadFile);
  elements.queryButton.addEventListener("click", submitQuery);
  elements.questionInput.addEventListener("input", () => {
    elements.questionLength.textContent = `${elements.questionInput.value.length} / 2000`;
    updateQueryButton();
  });
  elements.questionInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) submitQuery();
  });
  elements.suggestions.forEach((button) => button.addEventListener("click", () => {
    elements.questionInput.value = button.dataset.question || "";
    elements.questionInput.dispatchEvent(new Event("input", { bubbles: true }));
    elements.questionInput.focus();
  }));

  for (const eventName of ["dragenter", "dragover"]) {
    elements.dropZone.addEventListener(eventName, (event) => { event.preventDefault(); elements.dropZone.classList.add("dragging"); });
  }
  for (const eventName of ["dragleave", "drop"]) {
    elements.dropZone.addEventListener(eventName, (event) => { event.preventDefault(); elements.dropZone.classList.remove("dragging"); });
  }
  elements.dropZone.addEventListener("drop", (event) => selectFile(event.dataTransfer?.files?.[0]));
  document.addEventListener("dragover", (event) => event.preventDefault());
  document.addEventListener("drop", (event) => {
    if (!elements.dropZone.contains(event.target)) event.preventDefault();
  });

  checkConnection();
})();
