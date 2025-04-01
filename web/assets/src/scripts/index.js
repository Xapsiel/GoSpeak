import { initializeApp } from "./auth.js";

document.addEventListener("DOMContentLoaded", async () => {
    let isAuthenticated = await initializeApp();
    let container = document.getElementById("conference-container");

    if (isAuthenticated) {
        container.textContent = "Создать конференцию";
        container.onclick = () => {
            alert("Конференция создана!");
        };
    } else {
        container.textContent = "Зарегистрироваться";
        container.onclick = () => {
            window.location.href = "/register.html";
        };
    }
});