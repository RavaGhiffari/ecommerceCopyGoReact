import { useState } from "react";

interface User {
    id: number;
    name: string;
    email: string;
}

export function App() {
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    const fetchApiUsers = async () => {
        try {
            setLoading(true);
            setError("");

            const resp = await fetch("/api/users");

            if (!resp.ok) {
                throw new Error(`HTTP error: ${resp.status}`);
            }

            const data: User[] = await resp.json();

            setUsers(data);
        } catch (err) {
            setError(
                err instanceof Error ? err.message : "Unknown error"
            );
        } finally {
            setLoading(false);
        }
    };

    return (
        <>
            <button onClick={fetchApiUsers}>
                Fetch Users
            </button>

            {loading && <p>Loading...</p>}
            {error && <p>{error}</p>}

            <ul>
                {users.map((user) => (
                    <li key={user.id}>
                        {user.name} — {user.email}
                    </li>
                ))}
            </ul>
        </>
    );
}