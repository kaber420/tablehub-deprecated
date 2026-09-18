import { userManager } from "./auth.js";

const API_BASE_URL = import.meta.env.VITE_API_URL || "";

async function getAuthToken() {
    const loggedInUser = await userManager.getUser();
	return loggedInUser ? loggedInUser.id_token : "";
}

export async function apiFetch(path, options = {}) {
    const token = await getAuthToken();

    const headers = {
        "Content-Type": "application/json",
        ...options.headers,
    };

    if (token) {
        headers["Authorization"] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
        ...options,
        headers,
    });

    if (response.status === 401) {
        console.warn("API returned 401 Unauthorized, token might be expired.");
        // We throw or return the response to let caller handle redirect
    }

    return response;
}

// Bloqueo global de provisión para prevenir múltiples llamadas
let isProvisioning = false;

export async function createHub(orgID, payload = {}) {
    if (!orgID) throw new Error("ID de organización inválido");

    const response = await apiFetch(`/v1/orgs/${orgID}/hubs`, {
        method: "POST",
        body: JSON.stringify(payload)
    });

    if (!response.ok) {
        let errorMsg = "Error al crear Hub físico.";
        try {
            const text = await response.text();
            try {
                const body = JSON.parse(text);
                errorMsg = body.error || body.message || text;
            } catch {
                errorMsg = text || errorMsg;
            }
        } catch {}
        throw new Error(errorMsg);
    }

    return response.json();
}

export async function downloadHubProvision(orgID, hubID) {
    if (!orgID) throw new Error("ID de organización inválido");
    if (!hubID) throw new Error("ID de Hub inválido");
    if (isProvisioning) throw new Error("Ya hay una descarga en progreso");

    isProvisioning = true;
    try {
        const response = await apiFetch(`/v1/orgs/${orgID}/hubs/${hubID}/provision`, {
            method: "POST",
        });

        if (!response.ok) {
            let errorMsg = "Error al generar credenciales.";
            try {
                const text = await response.text();
                try {
                    const body = JSON.parse(text);
                    errorMsg = body.error || body.message || text;
                } catch {
                    errorMsg = text || errorMsg;
                }
            } catch {}
            throw new Error(errorMsg);
        }

        const blob = await response.blob();
        const disposition = response.headers.get("Content-Disposition");
        let filename = `hub-${hubID.substring(0, 8)}.thub`;

        if (disposition && disposition.includes("filename=")) {
            const matches = disposition.match(/filename="?([^"]+)"?/);
            if (matches && matches[1]) {
                filename = matches[1].replace(/[^a-zA-Z0-9.-_]/g, '');
            }
        }

        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        setTimeout(() => {
            window.URL.revokeObjectURL(url);
        }, 1000);
        
        return filename;
    } finally {
        isProvisioning = false;
    }
}

export async function fetchOrgStats(orgID) {
    if (!orgID) throw new Error("ID de organización inválido");
    const response = await apiFetch(`/v1/orgs/${orgID}/stats`);
    if (!response.ok) {
        throw new Error("Error al obtener estadísticas");
    }
    return response.json();
}

export async function fetchBranches(orgID) {
    if (!orgID) throw new Error("ID de organización inválido");
    const response = await apiFetch(`/v1/orgs/${orgID}/branches`);
    if (!response.ok) {
        throw new Error("Error al obtener sucursales");
    }
    return response.json();
}

export async function createBranch(orgID, payload) {
    if (!orgID) throw new Error("ID de organización inválido");
    const response = await apiFetch(`/v1/orgs/${orgID}/branches`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
    });
    if (!response.ok) {
        throw new Error("Error al crear sucursal");
    }
    return response.json();
}

export async function updateHub(orgID, hubID, payload) {
    if (!orgID || !hubID) throw new Error("IDs inválidos");
    const response = await apiFetch(`/v1/orgs/${orgID}/hubs/${hubID}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
    });
    if (!response.ok) {
        throw new Error("Error al actualizar Hub");
    }
    return response.json();
}
