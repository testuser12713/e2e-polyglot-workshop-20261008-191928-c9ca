import { Navigate, Route, Routes } from "react-router-dom";

import RequireAuth from "./auth/RequireAuth";
import AppShell from "./layout/AppShell";
import AppointmentPage from "./pages/customer/AppointmentPage";
import StatusPage from "./pages/customer/StatusPage";
import DashboardPage from "./pages/workshop/DashboardPage";
import LoginPage from "./pages/workshop/LoginPage";
import OrderDetailPage from "./pages/workshop/OrderDetailPage";
import OrderListPage from "./pages/workshop/OrderListPage";

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path="/" element={<AppointmentPage />} />
        <Route path="/status" element={<StatusPage />} />
        <Route path="/werkstatt/login" element={<LoginPage />} />
        <Route
          path="/werkstatt"
          element={
            <RequireAuth>
              <DashboardPage />
            </RequireAuth>
          }
        />
        <Route
          path="/werkstatt/auftraege"
          element={
            <RequireAuth>
              <OrderListPage />
            </RequireAuth>
          }
        />
        <Route
          path="/werkstatt/auftraege/:id"
          element={
            <RequireAuth>
              <OrderDetailPage />
            </RequireAuth>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
