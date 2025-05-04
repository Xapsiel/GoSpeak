import { initializeApp,logout } from "./auth.js";
const axiosInstance = axios.create({
    baseURL: `http://${domain}`,
});
function updateAuthUI(isAuthenticated) {
    const loginBtn = document.getElementById('loginBtn');
    const logoutBtn = document.getElementById('logoutBtn');

    if (loginBtn) {
        loginBtn.style.display = isAuthenticated ? 'none' : 'block';
    }
    if (logoutBtn) {
        logoutBtn.style.display = isAuthenticated ? 'block' : 'none';
    }
}

document.addEventListener("DOMContentLoaded", async () => {

    const isAuthenticated = await initializeApp();
    updateAuthUI(isAuthenticated);


    const logoutBtn = document.getElementById("logoutBtn");
    if (logoutBtn) {
        logoutBtn.addEventListener("click", async (event) => {
            event.preventDefault();
            try {
                await axiosInstance.post("/auth/logout");
                logout();
                updateAuthUI(false);
            } catch (error) {
                console.error("Ошибка при выходе:", error);
            }
        });
    }

});