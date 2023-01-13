import { createContext, FC, useContext, useState } from "react";
import Snackbar from "@mui/material/Snackbar/Snackbar";
import SnackbarContent from "@mui/material/SnackbarContent/SnackbarContent";
import Button from "@mui/material/Button/Button";
import { Stack, useMediaQuery } from "@mui/material";

interface INotification {
  message: string;
  type?: "info" | "error";
  actions?: Array<{
    name: string;
    onClick?(): void;
  }>;
}

interface ISendNotification {
  info(message: string, actions?: INotification["actions"]): void;
  error(message: string, actions?: INotification["actions"]): void;
}

interface INotificationContext {
  sendNotification: ISendNotification;
}
const NotificationContext = createContext<INotificationContext>({
  sendNotification: {
    info: () => null,
    error: () => null,
  },
});

export const NotificationProvider: FC = ({ children }) => {
  const useLightTheme = useMediaQuery("(prefers-color-scheme: light)");

  const [open, setOpen] = useState(false);
  const [values, setValues] = useState<INotification>({
    message: "",
    type: "info",
  });

  const sendNotification: ISendNotification = {
    info: (message, actions) => {
      setValues({ message, type: "info", actions });
      setOpen(true);
    },
    error: (message, actions) => {
      setValues({ message, type: "error", actions });
      setOpen(true);
    },
  };

  const DEFAULT_ACTION: INotification["actions"][0] = {
    name: "Dismiss",
    onClick: () => setOpen(false),
  };

  const buildActions = (actions: INotification["actions"] = []) => {
    return (
      <Stack spacing={1}>
        {actions.map((action) => (
          <Button
            key={action.name}
            variant="text"
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
              backgroundColor: (theme) => {
                if (values.type === "error") {
                  return useLightTheme
                    ? theme.palette.docker.red[100] // red-100 should be "#FDEAEA" but it's not 🤷‍♂️
                    : theme.palette.docker.red[200];
                }

                return useLightTheme
                  ? theme.palette.common.white
                  : theme.palette.docker.grey[200];
              },
              color: (theme) => theme.palette.text.primary,
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
