'use strict';
const loginForm=document.querySelector('#login-form'),loginError=document.querySelector('#login-error');
if(new URLSearchParams(location.search).has('error'))loginError.hidden=false;
if(new URLSearchParams(location.search).has('changed'))document.querySelector('#login-success').hidden=false;
loginForm.addEventListener('submit',async e=>{e.preventDefault();if(!UI.beginForm(loginForm,'Memeriksa login…'))return;loginError.hidden=true;try{const response=await UI.request('/login',{method:'POST',credentials:'same-origin',body:new URLSearchParams(UI.formData(loginForm))});if(response.status===429)throw Error('Terlalu banyak percobaan. Tunggu 10 menit, lalu coba lagi.');if(!response.ok)throw Error('Login belum dapat diproses. Coba kembali beberapa saat lagi.');const result=new URL(response.url);if(result.pathname==='/login')throw Error('Username atau password tidak sesuai.');UI.markClean(loginForm);location.assign('/')}catch(error){UI.formError(loginForm,error,loginError)}finally{UI.endForm(loginForm)}});
window.addEventListener('pageshow',()=>UI.endForm(loginForm));if(!loginError.hidden){loginError.setAttribute('role','alert');document.querySelector('#password').focus()}
