import { createContext, FC, useContext, useState } from "react";
import Snackbar from "@mui/material/Snackbar/Snackbar";
import SnackbarContent from "@mui/material/SnackbarContent/SnackbarContent";
import Button from "@mui/material/Button/Button";

export default function VackupSnackbar({ open, onClose }) {
  return <Snackbar open={open} onClose={onClose} />;
}

interface IValues {
  message: string;
  action?: {
    name: string;
    onClick(): void;
  };
}
interface INotificationContext {
  sendNotification(message: string, action?: IValues["action"]): void;
}
const NotificationContext = createContext<INotificationContext>({
  sendNotification: () => null,
});

export const NotificationProvider: FC = ({ children }) => {
  const [open, setOpen] = useState(false);
  const [values, setValues] = useState<IValues>({ message: "" });
  const sendNotification = (message: string, action?: IValues["action"]) => {
    setValues({ message, action });
    setOpen(true);
  };
  const DEFAULT_ACTION: IValues["action"] = {
    name: "Dismiss",
    onClick: () => setOpen(false),
  };

  const buildAction = (action: IValues["action"]) => (
    <Button
      onClick={() => {
        action.onClick();
        setOpen(false);
      }}
      size="small"
    >
      {action.name}
    </Button>
  );

  return (
    <NotificationContext.Provider value={{ sendNotification }}>
      {children}
      {open && (
        <Snackbar
          anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
          open={open}
          onClose={() => setOpen(false)}
        >
          <SnackbarContent
            message={values.message}
            action={buildAction(values.action || DEFAULT_ACTION)}
            sx={{
              ".MuiSnackbarContent-message": {
                maxWidth: "400px",
                wordWrap: "break-word",
              },
            }}
          />
        </Snackbar>
      )}
    </NotificationContext.Provider>
  );
};

export const useNotificationContext = () => useContext(NotificationContext);
