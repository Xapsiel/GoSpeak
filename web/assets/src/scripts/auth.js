import axios from 'https://cdn.jsdelivr.net/npm/axios/dist/esm/axios.min.js';

const elements = {
    signUpBtn: document.querySelector("#sign-upButton"),
    signInBtn: document.querySelector("#sign-inButton"),
};


const auth = {
    token: window.localStorage.getItem("jwtToken"),
    user: JSON.parse(window.localStorage.getItem("user") || null),
};

const axiosInstance = axios.create({
    baseURL: `http://${domain}`,
});

async function signIn(email, password) {
    try {
        const response = await axiosInstance.post("/auth/sign-in", { email, password });
        const { error, jwt_token: token, user } = response.data;

        if (token) {
            auth.token = token;
            auth.user = user;
            window.localStorage.setItem("jwtToken", token);
            window.localStorage.setItem("user", JSON.stringify(user));
            window.location.href = "/";
        } else {
            throw new Error(error || "No token received during sign-in.");
        }
    } catch (error) {
        console.error("Error during SignIn:", error);
        alert("Ошибка входа. Проверьте email и пароль.");
    }
}

async function signUp(name, email, password) {
    try {
        const response = await axiosInstance.post("/auth/sign-up", { name, email, password });
        if (!response.data.error) {
            await signIn(email, password);
        } else {
            alert(response.data.error || "Ошибка регистрации. Попробуйте снова.");
        }
    } catch (error) {
        console.error("Error during SignUp:", error);
        alert("Произошла ошибка при попытке регистрации.");
    }
}

async function isAuthenticated() {
    try {
        const response = await axiosInstance.get("/auth/validateToken", {
            headers: { "Authorization": `Bearer ${auth.token}` },
        });
        return response.data.isValid;
    } catch (error) {
        console.error("Ошибка проверки авторизации:", error);
        return false;
    }
}

function logout() {
    window.localStorage.removeItem("jwtToken");
    window.localStorage.removeItem("user");
    auth.token = null;
    auth.user = null;
    window.location.href = "/sign-in";
}

if (elements.signInBtn) {
    const loginForm = document.querySelector("#signInForm");
    if (loginForm) {
        loginForm.addEventListener("submit", async (event) => {
            event.preventDefault();
            const email = loginForm.querySelector("#email").value;
            const password = loginForm.querySelector("#password").value;
            await signIn(email, password);
        });
    }
}

if (elements.signUpBtn) {
    const signupForm = document.querySelector("#signUpForm");
    if (signupForm) {
        signupForm.addEventListener("submit", async (event) => {
            event.preventDefault();
            const name = signupForm.querySelector("#name").value;
            const email = signupForm.querySelector("#email").value;
            const password = signupForm.querySelector("#password").value;
            await signUp(name, email, password);
        });
    }
}

async function getUser() {
    try {
        const response = await axiosInstance.get("/user/", {
            headers: { "Authorization": `Bearer ${auth.token}` },
        });
        return response.data;
    } catch (error) {
        logout();
        return null;
    }
}

export async function initializeApp(){
    if (auth.token) {
        const user = await getUser();
        if (user) {
            auth.user = user;
            window.localStorage.setItem("user", JSON.stringify(user));
            return true;
        }
    }
    return  false;

}

document.addEventListener("DOMContentLoaded", () => {
    initializeApp();
});