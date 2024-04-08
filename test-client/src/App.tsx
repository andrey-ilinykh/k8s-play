import React, { useEffect, useState } from 'react';
import logo from './logo.svg';
import './App.css';


type ServerMessage = { server: string, counter: number, ns: string, router: string }

const isServerMessage = (m: any): m is ServerMessage => "server" in m && "ns" in m

type ConnectionState = { server: string, ns: string, counter: number, router: string, close: boolean }
type ConnectionStateSocket = { state: ConnectionState, socket: WebSocket }


const mkConnection = (cs: ConnectionState) => {
  const socket = new WebSocket('wss://www.mydomain.com/ws/' + cs.ns);
  socket.addEventListener('open', function (event) {
    console.log('connection made to server:', event);
  });

  socket.addEventListener('close', function (event) {
    console.log('connection closed:', event);
    socket.close();
  });

  socket.addEventListener('message', function (event) {
    const msg = JSON.parse(event.data)
    if (isServerMessage(msg)) {

      cs.counter = 0
      cs.server = msg.server
      cs.router = msg.router
      socket.send("{}")

    }
  }
  )
  return socket
}


const nsNumber = 32

const servers = ["backend-0", "backend-1", "backend-2", "backend-3",]
const nss = Array.from(Array(nsNumber).keys())


const model0: ConnectionStateSocket[] = nss.map(n => {
  let ns = "ns-" + n
  let state = {
    ns: ns,
    server: "",
    counter: 0,
    router: "",
    close: false
  }
  return {
    state: state,
    socket: mkConnection(state)
  }
}
)
function App() {
  const [model, setModel] = useState(model0)

  const updateModel = () => {

    model.forEach(css => {
      let cs = css.state
      cs.counter += 1
      if (cs.counter > 10) {
        css.socket.close()
        css.socket = mkConnection(cs)
      }
    })
    setModel(Object.assign([], model))
  }
  useEffect(() => {
    const interval = setInterval(() => { updateModel(); }, 1000);
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
  const renderCell = (idx: number, cs: ConnectionState, server: string) => {

    const mkText = (cs: ConnectionState) => {
      let arr = cs.router.split('-')
      let rid = arr.length === 3 ? arr[2] : "Bad router"
      if (cs.server === server)
        return rid

      return ""
    }
    const mkColor = (cs: ConnectionState) => {
      if (cs.counter < 2 && server === cs.server)
        return "lightgreen"
      if (cs.counter < 5 && server === cs.server)
        return "yellow"
      if (cs.counter >= 5 && server === cs.server)
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

  const renderRow = (idx: number, css: ConnectionStateSocket) => {

    let cs = css.state
    const close = () => {
      css.socket.close()
      css.socket = mkConnection(cs)
      cs.counter = 6
      setModel(Object.assign([], model))
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
        {servers.map(s => renderCell(idx, cs, s))}
        <td><button onClick={close}>Close</button></td>
      </tr>
    )
  }
  const renderTable = () => {

    return (
      <div className="App">

        <table>
          {renderHeader()}
          {nss.map(idx => renderRow(idx, model[idx]))}
        </table>
      </div>
    )
  }

  return renderTable()
}

export default App;
