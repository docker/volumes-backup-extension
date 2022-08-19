import { createContext, FC, useContext, useState } from "react";
import Snackbar from "@mui/material/Snackbar/Snackbar";
import SnackbarContent from "@mui/material/SnackbarContent/SnackbarContent";
import Button from "@mui/material/Button/Button";
import { Stack } from "@mui/material";

interface IValues {
  message: string;
  actions?: Array<{
    name: string;
    onClick?(): void;
  }>;
}
interface INotificationContext {
  sendNotification(message: string, action?: IValues["actions"]): void;
}
const NotificationContext = createContext<INotificationContext>({
  sendNotification: () => null,
});

export const NotificationProvider: FC<{}> = ({ children }) => {
  const [open, setOpen] = useState(false);
  const [values, setValues] = useState<IValues>({ message: "" });
  const sendNotification = (message: string, actions?: IValues["actions"]) => {
    setValues({ message, actions });
    setOpen(true);
  };
  const DEFAULT_ACTION: IValues["actions"][0] = {
    name: "Dismiss",
    onClick: () => setOpen(false),
  };

  const buildActions = (actions: IValues["actions"] = []) => {
    return (
      <Stack spacing={1}>
        {actions.map((action) => (
          <Button
            key={action.name}
            onClick={() => {
              if (action.onClick) action.onClick();
              setOpen(false);
            }}
            size="small"
          >
            {action.name}
          </Button>
        ))}
      </Stack>
    );
  };

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
            action={buildActions(values.actions || [DEFAULT_ACTION])}
            sx={{
              backgroundColor: (theme) => theme.palette.common.white,
              borderRadius: "4px !important",
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
