import React, { useState, useEffect } from 'react';
import {CartesianGrid, Line, LineChart, Tooltip, XAxis} from 'recharts';
import './App.css';
import {Transaction, Account} from './types';

interface DashboardData {
    accounts: Account[];
    transactions: Map<string,Transaction[]>;
}

function App() {
    const [data, setData] = useState<DashboardData>()
    useEffect(() => {
        api<DashboardData>("/api/pulse").then(d => setData(d))
    }, [])

    return (
        <div className="container max-w-screen-lg mx-auto mt-10">
            <p className="text-4xl font-extrabold">Budgie</p>
                {data?.transactions?.forEach((txs) => {
                    return (<LineChart
                            width={600}
                            height={400}
                            data={txs}
                        >
                            <XAxis dataKey="timestamp"/>
                            <Tooltip/>
                            <CartesianGrid stroke="#f5f5f5"/>
                            <Line type="monotone" dataKey="amount" stroke="#ff7300" yAxisId={0}/>
                        </LineChart>
                    )
                })
            }
        </div>
    );
}

function api<T>(url: string): Promise<T> {
    return fetch(url)
        .then(response => {
            if (!response.ok) {
                throw new Error(response.statusText)
            }
            return response.json()
        })
        .then((data) => data as T)
}


export default App;
