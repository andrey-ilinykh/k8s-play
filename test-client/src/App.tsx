import React, { useEffect, useState } from 'react';
import logo from './logo.svg';
import './App.css';
//import useWebSocket from 'react-use-websocket';

//const ws = new WebSocket('wss://www.mydomain.com/ws/ns-02')

type ServerMessage = { server: string, counter: number, ns: string, router: string }

const isServerMessage = (m: any): m is ServerMessage => "server" in m && "ns" in m

type ConnectionState = { server: string, ns: string, counter: number, router: string, close: boolean }

var connMap = new Map<number, ConnectionState>();

const createConnection = (cid: number) => () => {
  const ns = "ns-" + cid % 16
  const socket = new WebSocket('wss://www.mydomain.com/ws/' + ns);

  socket.addEventListener('open', function (event) {
    console.log('connection made to server:', event);
  });

  socket.addEventListener('close', function (event) {
    console.log('connection closed:', event);
    socket.close();
    setTimeout(createConnection(cid), 1000);  // reconnect here
  });

  socket.addEventListener('message', function (event) {
    const msg = JSON.parse(event.data)
    if (isServerMessage(msg)) {
      const sm = connMap.get(cid) || { server: msg.server, counter: 0, ns: msg.ns, router: msg.router, close: false };
      sm.counter = 0
      sm.server = msg.server
      sm.router = msg.router
      connMap = connMap.set(cid, sm)
      if (sm.close) {
        sm.close = false
        sm.counter = 200
        socket.close()
        createConnection(cid)
        return
      }
      socket.send("{}")
      // if ( msg.counter < 50){
      //   socket.send("{}")  

      // } else {
      //   sm.counter = 200
      // }


    }
    return socket
  }

  );

};
const nsNumber = 32

const servers = ["backend-0", "backend-1", "backend-2", "backend-3",]
const nss = Array.from(Array(nsNumber).keys())

nss.map(cid => createConnection(cid)())

function App() {

  const getConnMap = () => connMap
  const [time, setTime] = useState(Date.now());
  const updateConnMap = () => {

    getConnMap().forEach((ss, cid) => ss.counter < 100 ? ss.counter = ss.counter + 1 : ss.counter)
  }
  useEffect(() => {
    const interval = setInterval(() => { updateConnMap(); setTime(Date.now()) }, 1000);
    return () => {

      clearInterval(interval);
    };
  }, []);

  const renderHeader = () => {
    return (

      <tr>
        <th>#</th>
        <th>NS</th>
        {servers.map(s => (<th>{s}</th>))}

      </tr>)
  }
  const renderCell = (idx: number, server: string) => {

    let cs = connMap.get(idx)
    const mkText = (cs: ConnectionState) => {
      let arr = cs.router.split('-')
      let rid = arr.length === 3 ? arr[2] : "Bad router"
      if (cs.server == server)
        return rid
      // if(cs.counter < 2 && server === cs.server)
      //   return rid
      // if(cs.counter < 10 && server === cs.server)
      //   return rid

      // if(cs.counter > 100 && server === cs.server)
      //   return rid
      return ""
    }
    const mkColor = (cs: ConnectionState) => {
      if (cs.counter < 2 && server === cs.server)
        return "lightgreen"
      if (cs.counter < 10 && server === cs.server)
        return "yellow"
      if (cs.counter > 100 && server === cs.server)
        return "red"
      return "white"
    }
    let txt = cs ? mkText(cs) : "";
    let bgColor = cs ? mkColor(cs) : "white"

    return (<td style={{
      'backgroundColor': bgColor,
      'borderBottom': '1px solid black',
      'minWidth': '6em'
    }} >{txt}</td>)
  }

  const renderRow = (idx: number) => {
    const markClose = () => {
      let cs = getConnMap().get(idx)
      if (cs !== undefined) {
        cs.close = true
        cs.counter = 200
      }
    }
    return (
      <tr>
        <td style={
          {
            'borderBottom': '1px solid black',
            'minWidth': '6em'
          }
        } >{idx}</td>
        <td style={
          {
            'borderBottom': '1px solid black',
            'minWidth': '6em'
          }
        } >{"ns-" + idx % 16}</td>
        {servers.map(s => renderCell(idx, s))}
        <td><button onClick={markClose}>Close</button></td>
      </tr>
    )
  }
  const renderTable = () => {

    return (
      <div className="App">

        <table>
          {renderHeader()}
          {nss.map(idx => renderRow(idx))}
        </table>
      </div>
    )
  }

  return renderTable()
}

export default App;
