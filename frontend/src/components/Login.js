import React, { useState } from 'react';

export default function Login( props ) {
  const [username, setUsername] = useState();
  const [password, setPassword] = useState();
  const operation = props.isSignup ? "Sign up" : "Sign in"

  redirectIfLogged()

  const handleSubmit = async e => {
	console.log(props.isSignup, operation)
	e.preventDefault();
	if(props.isSignup) {
		await signup(username,password); 
	}else{
		await signin(username,password); 
	}
  }

  return(
	<div className="login">
	  <h1>{ operation }</h1>
	  <form onSubmit={handleSubmit}>
		<span><label>Login:</label>
		<input type="text" onChange={e => setUsername(e.target.value)}/></span>

		<span><label>Password:</label>
		<input type="password" onChange={e =>setPassword(e.target.value)}/></span>
		
		<button type="submit">{ operation }</button>
	  { props.isSignup ?  <a href='/signin'>Login instead</a> : <a href='/signup'>Create new account</a> }
		
	  </form>
	</div>
  )

  function redirectIfLogged(){
	let cookies = document.cookie
	if (cookies.includes("session_token")) window.location = "/"
  }

  async function signin(username,password) {
	fetch('/signin', {
	  method: 'POST',
	  credentials: "same-origin",
	  headers: {
		'content-type': 'application/json',
	  },
	  body: JSON.stringify({username: username, password: password})
	}).then(response => {
		const isJson = response.headers.get('content-type')?.includes('application/json');
        const data = isJson ? response.json() : null;

		if(!response.ok){
			if (response.status == 401){
				alert("Wrong login or password")
				return
			}
			const error = (data && data.message) || response.status;
            return Promise.reject(error);
		}
		window.location = "/";
	}).catch(error => {
	  console.log(error)
	});
  }

  async function signup(username,password) {
	fetch('/signup', {
	  method: 'POST',
	  credentials: "same-origin",
	  headers: {
		'content-type': 'application/json',
	  },
	  body: JSON.stringify({username: username, password: password})
	}).then(response => {
		const isJson = response.headers.get('content-type')?.includes('application/json');
        const data = isJson ? response.json() : null;

		if(!response.ok){
			if (response.status == 409){
				alert("Username already exists")
				return
			}
			const error = (data && data.message) || response.status;
            return Promise.reject(error);
		}
		window.location = "/signin";
	}).catch(error => {
	  console.log(error)
	});
  }
}
