import React, { useState } from 'react';
import { BrowserRouter, Route, Routes} from 'react-router-dom'
import  Files  from './components/Filebrowser'
import Login from './components/Login'

import 'font-awesome/css/font-awesome.min.css'

import './App.css';

function App() {
  return (
    <div className="App background-gradient">
	  <div className="flex">
		<BrowserRouter>
		  <Routes>
			<Route path='/' element={<Files/>}/>
			<Route path='/signin' element={<Login isSignup={false} />}/>
			<Route path='/signup' element={<Login isSignup={true}/>}/>
		  </Routes>
		</BrowserRouter>
	  </div>
    </div>
  );
}

export default App;
