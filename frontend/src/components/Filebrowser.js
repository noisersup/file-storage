import React, { useEffect,useState,useRef}  from 'react'
import Moment from 'moment'

import { DndProvider } from 'react-dnd'
import { HTML5Backend } from 'react-dnd-html5-backend'

import '../../node_modules/react-keyed-file-browser/dist/react-keyed-file-browser.css';
import { RawFileBrowser, Icons } from 'react-keyed-file-browser'

export default function Files () {
	const [files, setFiles] = useState([]);
	const [lastRefresh, setLastRefresh] = useState(0)
	let filebrowser = useRef();

	useEffect(() => {
	  getFiles("")
	},[])

	const getPath = () => {
	  let path = ""
	  let selection = filebrowser.current.state.selection
	  if(selection.length >0){
		let selected = filebrowser.current.state.selection[0]
		console.log("selected:",selected)
		let slashIndex = selected.lastIndexOf('/')
		if(slashIndex < 0) return path // '/' not found	

		path="/"

		if(slashIndex == selected.length-1) return path + selected.slice(0,-1)

		path += selected.slice(0,slashIndex)
	  }
	  return path
	}

	const addDirectory = () => {
	  let name = document.getElementById("newdir-name").value;
	  refreshAuth()
	  fetch('/drive'+getPath()+"/"+name,{
		method: "POST",
	  }).then(console.log); 
	  alert('The file has been uploaded successfully.');
	}

	const uploadFile = () => {
	  refreshAuth()

	  let fileupload = document.getElementById("fileupload");
	  let formData = new FormData();
	  formData.append("file",fileupload.files[0]);
	  fetch('/drive'+getPath(), {
		  method: "POST", 
		  body: formData
	  }).then(console.log); 
	  alert('The file has been uploaded successfully.');
	};

	const logout = () => {
	  fetch('/logout', {
		  method: "POST", 
	  }).then(console.log); 
		delCookie("session_token")
		console.log("signin")
		window.location = "/signin"
	}

	const delCookie = (name) => {
  		document.cookie = name +'=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT;';
	}


	const handleRenameFolder = (oldKey, newKey) => {
		this.setState(state => {
			const newFiles = [];
			state.files.map((file) => {
				if(file.key.substr(0, oldKey.length) === oldKey) {
					newFiles.push({
						...file,
						key: file.key.replace(oldKey,newKey),
						modified: +Moment(),
					})
				} else {
					newFiles.push(file)
				}
			})
			state.files = newFiles
			return state
		})
	}


	const handleOpenFolder = (file) => {
	  /*
		const curr = this.Filebrowser.current;
		console.log(curr)
	  */
		if (!areChildLoaded(file.key)) {
			getFiles(file.key.slice(0,file.key.length-1));
		}
	}

	const handleOpenFile = (file) => {
		console.log(file)
		downloadFile(file.key)
	}


	
	const pushFileTo = (files, file) => {
	  for(let i=0;i<files.length;i++){
		let item = files[i];
		if(file.key == item.key) {
		  item = file;
		  return 
		}
	  }
	  files.push(file)
	}

	const areChildLoaded = (path) =>{
		let pattern = "^"+path+".+";

		for(let i=0;i<files.length;i++){
		let item = files[i];
			let match = item.key.match(pattern);
			if(match != null && match.length >0){
				return true
			}
		}
		return false
	}

	const downloadFile = (path) => {

	  	refreshAuth()
		let location = window.location
		window.open(location.protocol+"//"+location.hostname+":8000/drive/"+path, "_blank")
	}

	const refreshAuth = () => {
		if (Date.now()/1000 - lastRefresh < 20) return
		fetch('/refresh', {
		  method: 'POST',
		  credentials: "same-origin",
		}).then(response => {
			if(!response.ok){
				if (response.status == 401) {
					console.log("signin")
					window.location = "/signin"
					delCookie("session_token")
					return
				}
			}
			setLastRefresh(Date.now()/1000)
		}).catch(error => {
		  console.log(error)
		});
	  }


	const getFiles = (dir) => {
		console.log(dir)
		dir = dir.trim();
		let fetchEndpoint = "/drive";
		if(dir != undefined && dir.length > 0){
		  fetchEndpoint += "/"+dir
		  dir = dir + "/";
		}
		 
	  	refreshAuth()
		fetch(fetchEndpoint, {
		  method: 'GET',
		  credentials: "same-origin",
		}).then( async response => {
		  const isJson = response.headers.get('content-type')?.includes('application/json');
		  const data = isJson ? await response.json() : null;

		  if(!response.ok){
			  const error = (data && data.message) || response.status;
			  if (response.status == 401) {
				console.log("signin")
				window.location = "/signin"
				return
			  }
			  console.error(data)
			  return Promise.reject(error);
		  }
		  let newFiles = files;
		  data.files.forEach((item) => {
			let path = dir+item.name;
			if(item.isDirectory) path +="/"
			//this.pushFileTo(newFiles,{key: path, modified: +Moment().subtract(1, 'hours'), size: 1.5 * 1024 * 1024})
			pushFileTo(newFiles,{key: path})
		  });
		  setFiles([...newFiles]);
		})
		}


	return (
	  <div className="filestorage">
		<header>
			<span><button id="logout" onClick={logout}> Logout </button></span>
			<div className="upload">
			  <input id="fileupload" type="file" name="fileupload" /> 
			  <button id="upload-button" onClick={uploadFile}> Upload </button>
			</div>
			<div className="upload">
			  <input id="newdir-name" type="text" name="name" /> 
			  <button id="newdir-button" onClick={addDirectory}> Add directory </button>
			</div>
		</header>
		<DndProvider backend={HTML5Backend}>
			<RawFileBrowser
			  ref={filebrowser}
			  files={files}
			  icons={Icons.FontAwesome(4)}
			  canFilter={false}
			  detailRenderer={() => <></>}
			  headerRenderer={() => null}
			  //actionRenderer={ActionRenderer} 
			  onCreateFiles={console.log}
			  onCreateFolers={console.log}

			  onRenameFolder={handleRenameFolder}
			  onMoveFolder={handleRenameFolder}

			  onSelectFolder={handleOpenFolder}

			  //onDownloadFile={this.handleOpenFile}

			  onSelectFile={handleOpenFile}
			/>
		</DndProvider>
	  </div>
	)
}
