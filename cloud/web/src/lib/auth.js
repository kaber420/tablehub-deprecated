import { UserManager, WebStorageStateStore } from "oidc-client-ts";
import { writable } from "svelte/store";

export const authProvider = import.meta.env.VITE_AUTH_PROVIDER || "logto";
const authority = import.meta.env.VITE_OIDC_ISSUER || "http://localhost:3002/oidc";
const client_id = import.meta.env.VITE_OIDC_CLIENT_ID || "";

const scope = authProvider === "logto" 
    ? "openid profile email urn:logto:scope:organizations urn:logto:scope:organization_roles" 
    : "openid profile email";

const oidcConfig = {
    authority,
    client_id,
    redirect_uri: "http://localhost:5173/auth/callback",
    post_logout_redirect_uri: "http://localhost:5173/",
    response_type: "code",
    scope,
    extraQueryParams: {
        resource: "urn:logto:resource:organizations"
    },
    userStore: new WebStorageStateStore({ store: window.sessionStorage }),
};

export const userManager = new UserManager(oidcConfig);
export const user = writable(null);
export const isAuthenticated = writable(false);

export async function login() {
    await userManager.signinRedirect();
}

export async function logout() {
    await userManager.signoutRedirect();
}

export async function initAuth() {
    try {
        const loggedInUser = await userManager.getUser();
        if (loggedInUser && !loggedInUser.expired) {
            user.set(loggedInUser);
            isAuthenticated.set(true);
        }
    } catch (e) {
        console.error("Auth init error:", e);
    }
}

