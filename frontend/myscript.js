function Login(username,password) {
	let opts = {
		method: "POST",
		body: JSON.stringify( {username: username, password: password } )
	};
	fetch( 'http://localhost:8000/signin', opts)
		//.then( resp => resp.json() )
		.then( resp => {
			console.log(resp)
		})
}


function GetFile(path){
	fetch (`http://localhost:8000/drive/${path}`)
		.then( resp => resp.json() )
		.then( resp => {
			console.log(resp);
		}); 
}
